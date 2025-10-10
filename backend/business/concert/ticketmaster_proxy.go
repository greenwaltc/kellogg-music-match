package concert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type TicketmasterProxy struct {
	config     *config.TicketmasterConfig
	httpClient *http.Client
}

// Context keys used to pin a time window across a sync cycle
type tmCtxKey string

const (
	tmStartKey tmCtxKey = "tm_start_utc"
	tmEndKey   tmCtxKey = "tm_end_utc"
)

// TicketmasterEvent represents a single event from the API
type TicketmasterEvent struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	URL   string `json:"url"`
	Dates struct {
		Start struct {
			LocalDate string `json:"localDate"`
			LocalTime string `json:"localTime"`
		} `json:"start"`
	} `json:"dates"`
	Embedded struct {
		Venues []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Address struct {
				Line1 string `json:"line1"`
			} `json:"address"`
			City struct {
				Name string `json:"name"`
			} `json:"city"`
		} `json:"venues"`
		Attractions []struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Classifications []struct {
				Segment struct {
					Name string `json:"name"`
				} `json:"segment"`
				Genre struct {
					Name string `json:"name"`
				} `json:"genre"`
			} `json:"classifications"`
		} `json:"attractions"`
	} `json:"_embedded"`
	PriceRanges []struct {
		Type string  `json:"type"`
		Min  float64 `json:"min"`
		Max  float64 `json:"max"`
	} `json:"priceRanges"`
}

// TicketmasterResponse represents the API response structure
type TicketmasterResponse struct {
	Embedded struct {
		Events []TicketmasterEvent `json:"events"`
	} `json:"_embedded"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

// NewTicketmasterProxy creates a new instance of TicketmasterProxy
func NewTicketmasterProxy(cfg *config.TicketmasterConfig) *TicketmasterProxy {
	timeout := time.Duration(cfg.Timeout) * time.Second
	return &TicketmasterProxy{
		config:     cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// FetchConcerts fetches concerts from Ticketmaster API for Chicago area in the configured date range
func (p *TicketmasterProxy) FetchConcerts(ctx context.Context) (*TicketmasterResponse, error) {
	return p.FetchConcertsWithPagination(ctx, 0)
}

// FetchConcertsWithPagination fetches concerts with pagination support
func (p *TicketmasterProxy) FetchConcertsWithPagination(ctx context.Context, page int) (*TicketmasterResponse, error) {
	tr := otel.Tracer("external.ticketmaster")
	ctx, span := tr.Start(ctx, "FetchConcertsWithPagination")
	span.SetAttributes(attribute.Int("app.page", page))
	defer span.End()
	// Calculate date range; prefer pinned window from context when present
	// Use UTC for API query parameters
	start := time.Now().UTC()
	if v := ctx.Value(tmStartKey); v != nil {
		if t, ok := v.(time.Time); ok {
			start = t.UTC()
		}
	}
	// End time from context, else derive from config with clamp
	var endDate time.Time
	if v := ctx.Value(tmEndKey); v != nil {
		if t, ok := v.(time.Time); ok {
			endDate = t.UTC()
		}
	}
	if endDate.IsZero() {
		months := p.config.DateRangeMonths
		if months <= 0 || months > 12 {
			if months > 12 {
				span.SetAttributes(attribute.Int("ticketmaster.months_clamped", months))
			}
			months = 12
		}
		endDate = start.AddDate(0, months, 0)
	}

	// Build query parameters
	params := url.Values{}
	params.Set("apikey", p.config.ConsumerKey)
	// Prefer segmentName filter, which is widely supported
	params.Set("segmentName", "Music") // Only music events
	params.Set("startDateTime", start.Format("2006-01-02T15:04:05Z"))
	params.Set("endDateTime", endDate.Format("2006-01-02T15:04:05Z"))
	params.Set("size", fmt.Sprintf("%d", p.config.MaxResults)) // Configurable max results per page
	params.Set("page", fmt.Sprintf("%d", page))                // Requested page
	params.Set("sort", "date,asc")                             // Sort by date ascending
	params.Set("locale", "*")                                  // Avoid locale filtering

	// Add useful telemetry context (avoid logging API key or full URL)
	span.SetAttributes(
		attribute.String("ticketmaster.startDateTime", start.Format(time.RFC3339)),
		attribute.String("ticketmaster.endDateTime", endDate.Format(time.RFC3339)),
		attribute.Int("ticketmaster.size", p.config.MaxResults),
		attribute.Int("ticketmaster.page_request", page),
		attribute.String("ticketmaster.segment", "Music"),
		attribute.String("ticketmaster.locale", "*"),
	)

	// Validate geo lat/long if provided
	geoValid := false
	if p.config.GeoLatLong != "" {
		// Accept simple "lat,long" where each is -?digits(.digits)?
		re := regexp.MustCompile(`^-?\d{1,3}(?:\.\d+)?,-?\d{1,3}(?:\.\d+)?$`)
		geoValid = re.MatchString(p.config.GeoLatLong)
		if !geoValid {
			span.SetAttributes(attribute.String("ticketmaster.geo_invalid", p.config.GeoLatLong))
		}
	}

	// If geo radius search configured and valid, prefer that over city/state
	// track selected mode for logging
	mode := "city"
	if geoValid && p.config.Radius > 0 {
		// Ticketmaster Discovery API expects 'latlong' for geospatial search
		params.Set("latlong", p.config.GeoLatLong)
		params.Set("radius", fmt.Sprintf("%d", p.config.Radius))
		if p.config.RadiusUnit != "" {
			params.Set("unit", p.config.RadiusUnit)
		}
		mode = "geo"
		span.SetAttributes(
			attribute.String("ticketmaster.mode", "geo"),
			attribute.String("ticketmaster.geo", p.config.GeoLatLong),
			attribute.Int("ticketmaster.radius", p.config.Radius),
			attribute.String("ticketmaster.unit", p.config.RadiusUnit),
		)
	} else {
		// Fallback to legacy city/state search
		params.Set("city", p.config.DefaultCity)
		params.Set("stateCode", p.config.DefaultState)
		params.Set("countryCode", p.config.DefaultCountry)
		mode = "city"
		span.SetAttributes(
			attribute.String("ticketmaster.mode", "city"),
			attribute.String("ticketmaster.city", p.config.DefaultCity),
			attribute.String("ticketmaster.state", p.config.DefaultState),
		)
	}

	// Build the full URL
	// Use .json endpoint to explicitly request JSON responses
	fullURL := fmt.Sprintf("%s/events.json?%s", p.config.BaseURL, params.Encode()) // Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "KelloggMusicMatch/1.0")

	// Make the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		span.SetStatus(codes.Error, fmt.Sprintf("status %d", resp.StatusCode))
		// Read up to 2KB of body for diagnostics and try to parse JSON error fields
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		var bodySnippet struct {
			Detail  string `json:"detail"`
			Message string `json:"message"`
			Error   string `json:"error"`
			Errors  any    `json:"errors"`
		}
		_ = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&bodySnippet)
		// attach a trimmed body preview to span for visibility (capped to 512 chars)
		preview := string(bodyBytes)
		if len(preview) > 512 {
			preview = preview[:512]
		}
		span.SetAttributes(attribute.String("ticketmaster.error_body", preview))
		// structured log for quick diagnosis
		lg := logger.FromCtx(ctx)
		fields := []any{
			"status", resp.StatusCode,
			"page", page,
			"size", p.config.MaxResults,
			"mode", mode,
			"start", start.Format(time.RFC3339),
			"end", endDate.Format(time.RFC3339),
			"segment", "Music",
			"locale", "*",
		}
		if mode == "geo" {
			fields = append(fields, "geo", p.config.GeoLatLong, "radius", p.config.Radius, "unit", p.config.RadiusUnit)
		} else {
			fields = append(fields, "city", p.config.DefaultCity, "state", p.config.DefaultState, "country", p.config.DefaultCountry)
		}
		fields = append(fields, "error_body", preview)
		lg.Warn("ticketmaster non-200 response", fields...)
		if bodySnippet.Detail != "" || bodySnippet.Message != "" || bodySnippet.Error != "" {
			msg := fmt.Sprintf("%s %s %s", bodySnippet.Detail, bodySnippet.Message, bodySnippet.Error)
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(msg))
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(preview))
	}

	// Parse JSON response
	var tmResponse TicketmasterResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmResponse); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	span.SetAttributes(
		attribute.Int("app.events", len(tmResponse.Embedded.Events)),
		attribute.Int("ticketmaster.page_number", tmResponse.Page.Number),
		attribute.Int("ticketmaster.page_totalPages", tmResponse.Page.TotalPages),
		attribute.Int("ticketmaster.page_totalElements", tmResponse.Page.TotalElements),
	)

	return &tmResponse, nil
}

// FetchConcertsByArtist fetches concerts for a specific artist in Chicago
func (p *TicketmasterProxy) FetchConcertsByArtist(ctx context.Context, artistName string) (*TicketmasterResponse, error) {
	tr := otel.Tracer("external.ticketmaster")
	ctx, span := tr.Start(ctx, "FetchConcertsByArtist")
	span.SetAttributes(attribute.String("app.artist", artistName))
	defer span.End()
	start := time.Now().UTC()
	if v := ctx.Value(tmStartKey); v != nil {
		if t, ok := v.(time.Time); ok {
			start = t.UTC()
		}
	}
	var endDate time.Time
	if v := ctx.Value(tmEndKey); v != nil {
		if t, ok := v.(time.Time); ok {
			endDate = t.UTC()
		}
	}
	if endDate.IsZero() {
		months := p.config.DateRangeMonths
		if months <= 0 || months > 12 {
			if months > 12 {
				span.SetAttributes(attribute.Int("ticketmaster.months_clamped", months))
			}
			months = 12
		}
		endDate = start.AddDate(0, months, 0)
	}

	params := url.Values{}
	params.Set("apikey", p.config.ConsumerKey)
	params.Set("segmentName", "Music")
	params.Set("keyword", artistName) // Search for specific artist
	params.Set("startDateTime", start.Format("2006-01-02T15:04:05Z"))
	params.Set("endDateTime", endDate.Format("2006-01-02T15:04:05Z"))
	params.Set("size", "50") // Could make this configurable too
	params.Set("sort", "date,asc")
	params.Set("locale", "*")

	span.SetAttributes(
		attribute.String("ticketmaster.startDateTime", start.Format(time.RFC3339)),
		attribute.String("ticketmaster.endDateTime", endDate.Format(time.RFC3339)),
		attribute.Int("ticketmaster.size", 50),
		attribute.String("ticketmaster.segment", "Music"),
		attribute.String("ticketmaster.locale", "*"),
		attribute.String("ticketmaster.keyword", artistName),
	)

	geoValid := false
	if p.config.GeoLatLong != "" {
		re := regexp.MustCompile(`^-?\d{1,3}(?:\.\d+)?,-?\d{1,3}(?:\.\d+)?$`)
		geoValid = re.MatchString(p.config.GeoLatLong)
		if !geoValid {
			span.SetAttributes(attribute.String("ticketmaster.geo_invalid", p.config.GeoLatLong))
		}
	}

	if geoValid && p.config.Radius > 0 {
		params.Set("latlong", p.config.GeoLatLong)
		params.Set("radius", fmt.Sprintf("%d", p.config.Radius))
		if p.config.RadiusUnit != "" {
			params.Set("unit", p.config.RadiusUnit)
		}
		span.SetAttributes(
			attribute.String("ticketmaster.mode", "geo"),
			attribute.String("ticketmaster.geo", p.config.GeoLatLong),
			attribute.Int("ticketmaster.radius", p.config.Radius),
			attribute.String("ticketmaster.unit", p.config.RadiusUnit),
		)
	} else {
		params.Set("city", p.config.DefaultCity)
		params.Set("stateCode", p.config.DefaultState)
		params.Set("countryCode", p.config.DefaultCountry)
		span.SetAttributes(
			attribute.String("ticketmaster.mode", "city"),
			attribute.String("ticketmaster.city", p.config.DefaultCity),
			attribute.String("ticketmaster.state", p.config.DefaultState),
		)
	}

	fullURL := fmt.Sprintf("%s/events.json?%s", p.config.BaseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "KelloggMusicMatch/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		span.SetStatus(codes.Error, fmt.Sprintf("status %d", resp.StatusCode))
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		var bodySnippet struct {
			Detail  string `json:"detail"`
			Message string `json:"message"`
			Error   string `json:"error"`
			Errors  any    `json:"errors"`
		}
		_ = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&bodySnippet)
		preview := string(bodyBytes)
		if len(preview) > 512 {
			preview = preview[:512]
		}
		span.SetAttributes(attribute.String("ticketmaster.error_body", preview))
		lg := logger.FromCtx(ctx)
		lg.Warn("ticketmaster non-200 response (artist)", "status", resp.StatusCode, "keyword", artistName, "start", start.Format(time.RFC3339), "end", endDate.Format(time.RFC3339), "error_body", preview)
		if bodySnippet.Detail != "" || bodySnippet.Message != "" || bodySnippet.Error != "" {
			msg := fmt.Sprintf("%s %s %s", bodySnippet.Detail, bodySnippet.Message, bodySnippet.Error)
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(msg))
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(preview))
	}

	var tmResponse TicketmasterResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmResponse); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	span.SetAttributes(
		attribute.Int("app.events", len(tmResponse.Embedded.Events)),
		attribute.Int("ticketmaster.page_number", tmResponse.Page.Number),
		attribute.Int("ticketmaster.page_totalPages", tmResponse.Page.TotalPages),
		attribute.Int("ticketmaster.page_totalElements", tmResponse.Page.TotalElements),
	)

	return &tmResponse, nil
}

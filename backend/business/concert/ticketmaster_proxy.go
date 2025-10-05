package concert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type TicketmasterProxy struct {
	config     *config.TicketmasterConfig
	httpClient *http.Client
}

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
	// Calculate date range based on configuration (default 6 months)
	now := time.Now()
	months := p.config.DateRangeMonths
	if months <= 0 {
		months = 6 // Default to 6 months if not configured
	}
	endDate := now.AddDate(0, months, 0)

	// Build query parameters
	params := url.Values{}
	params.Set("apikey", p.config.ConsumerKey)
	params.Set("classificationName", "music") // Only music events
	params.Set("startDateTime", now.Format("2006-01-02T15:04:05Z"))
	params.Set("endDateTime", endDate.Format("2006-01-02T15:04:05Z"))
	params.Set("size", fmt.Sprintf("%d", p.config.MaxResults)) // Configurable max results per page
	params.Set("page", fmt.Sprintf("%d", page))                // Requested page
	params.Set("sort", "date,asc")                             // Sort by date ascending
	params.Set("includeSpellcheck", "yes")                     // Include spell suggestions

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
	if geoValid && p.config.Radius > 0 {
		params.Set("geoPoint", p.config.GeoLatLong)
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
		// Fallback to legacy city/state search
		params.Set("city", p.config.DefaultCity)
		params.Set("stateCode", p.config.DefaultState)
		params.Set("countryCode", p.config.DefaultCountry)
		span.SetAttributes(
			attribute.String("ticketmaster.mode", "city"),
			attribute.String("ticketmaster.city", p.config.DefaultCity),
			attribute.String("ticketmaster.state", p.config.DefaultState),
		)
	}

	// Build the full URL
	fullURL := fmt.Sprintf("%s/events?%s", p.config.BaseURL, params.Encode()) // Create HTTP request
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
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse JSON response
	var tmResponse TicketmasterResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmResponse); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	span.SetAttributes(attribute.Int("app.events", len(tmResponse.Embedded.Events)))

	return &tmResponse, nil
}

// FetchConcertsByArtist fetches concerts for a specific artist in Chicago
func (p *TicketmasterProxy) FetchConcertsByArtist(ctx context.Context, artistName string) (*TicketmasterResponse, error) {
	tr := otel.Tracer("external.ticketmaster")
	ctx, span := tr.Start(ctx, "FetchConcertsByArtist")
	span.SetAttributes(attribute.String("app.artist", artistName))
	defer span.End()
	now := time.Now()
	months := p.config.DateRangeMonths
	if months <= 0 {
		months = 6 // Default to 6 months if not configured
	}
	endDate := now.AddDate(0, months, 0)

	params := url.Values{}
	params.Set("apikey", p.config.ConsumerKey)
	params.Set("classificationName", "music")
	params.Set("keyword", artistName) // Search for specific artist
	params.Set("startDateTime", now.Format("2006-01-02T15:04:05Z"))
	params.Set("endDateTime", endDate.Format("2006-01-02T15:04:05Z"))
	params.Set("size", "50") // Could make this configurable too
	params.Set("sort", "date,asc")

	geoValid := false
	if p.config.GeoLatLong != "" {
		re := regexp.MustCompile(`^-?\d{1,3}(?:\.\d+)?,-?\d{1,3}(?:\.\d+)?$`)
		geoValid = re.MatchString(p.config.GeoLatLong)
		if !geoValid {
			span.SetAttributes(attribute.String("ticketmaster.geo_invalid", p.config.GeoLatLong))
		}
	}

	if geoValid && p.config.Radius > 0 {
		params.Set("geoPoint", p.config.GeoLatLong)
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

	fullURL := fmt.Sprintf("%s/events?%s", p.config.BaseURL, params.Encode())

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
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var tmResponse TicketmasterResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmResponse); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	span.SetAttributes(attribute.Int("app.events", len(tmResponse.Embedded.Events)))

	return &tmResponse, nil
}

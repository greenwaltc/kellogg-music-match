package business

import (
	"context"
	"net/http"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// GeneratedEventsAdapter adapts EventSearchService to the generated.EventsAPIServicer interface
type GeneratedEventsAdapter struct {
	svc *EventSearchService
	cfg *config.Config
}

func NewGeneratedEventsAdapter(svc *EventSearchService, cfg *config.Config) *GeneratedEventsAdapter {
	return &GeneratedEventsAdapter{svc: svc, cfg: cfg}
}

// SearchEvents implements GET /events/search via on-demand Ticketmaster Discovery
func (a *GeneratedEventsAdapter) SearchEvents(ctx context.Context, keyword, segmentName, classificationName, countryCode, stateCode, city, latlong string, radius int32, startDateTime, endDateTime time.Time, sort string, size, page int32, includeAssociated bool) (generated.ImplResponse, error) {
	if !a.cfg.Ticketmaster.OnDemand {
		// Not enabled; return 404 to mirror temporary behavior
		return generated.Response(http.StatusNotFound, generated.ErrorResponse{Message: "on-demand events not enabled", CreatedAt: time.Now().UTC()}), nil
	}
	// Map inputs for the service
	var startPtr, endPtr *time.Time
	if !startDateTime.IsZero() {
		t := startDateTime
		startPtr = &t
	}
	if !endDateTime.IsZero() {
		t := endDateTime
		endPtr = &t
	}
	in := SearchInput{
		Keyword:            keyword,
		SegmentName:        segmentName,
		ClassificationName: classificationName,
		CountryCode:        countryCode,
		StateCode:          stateCode,
		City:               city,
		LatLong:            latlong,
		Radius:             int(radius),
		StartDateTime:      startPtr,
		EndDateTime:        endPtr,
		Sort:               sort,
		Size:               int(size),
		Page:               int(page),
		IncludeAssociated:  includeAssociated,
	}
	tmResp, err := a.svc.Search(ctx, in)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: err.Error(), CreatedAt: time.Now().UTC()}), nil
	}
	// Build EventsPage
	items := make([]generated.Event, 0, len(tmResp.Embedded.Events))
	for _, e := range tmResp.Embedded.Events {
		// Parse time (fallback to now on parse error)
		startStr := ""
		if e.Dates.Start.LocalDate != "" {
			startStr = e.Dates.Start.LocalDate + "T" + e.Dates.Start.LocalTime + "Z"
		}
		startUTC := time.Now().UTC()
		if startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				startUTC = t
			}
		}
		// Venue and location (provide safe defaults to satisfy required fields)
		venueName := "Unknown Venue"
		cityName := ""
		stateCode := ""
		countryCode := "US"
		if len(e.Embedded.Venues) > 0 {
			v := e.Embedded.Venues[0]
			if v.Name != "" {
				venueName = v.Name
			}
			cityName = v.City.Name
		}

		ev := generated.Event{
			ExternalId: e.ID,
			Source:     "ticketmaster",
			Name:       e.Name,
			StartUtc:   startUTC,
			Url:        e.URL,
			Venue: generated.Venue{
				Name: venueName,
				// Address optional; we can enrich later
			},
			Location:    generated.EventLocation{City: cityName, State: stateCode, Country: countryCode},
			Association: generated.EventAssociation{InterestedCount: 0, GoingCount: 0, LfgCount: 0},
		}
		items = append(items, ev)
	}
	pageResp := generated.EventsPage{Page: int32(in.Page), Size: int32(in.Size), Total: int32(tmResp.Page.TotalElements), Items: items}
	return generated.Response(http.StatusOK, pageResp), nil
}

// GetAssociatedEvents is not yet implemented in Phase 1
func (a *GeneratedEventsAdapter) GetAssociatedEvents(ctx context.Context, startDateTime, endDateTime time.Time, segmentName, city string, size, page int32) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNotImplemented, generated.ErrorResponse{Message: "not implemented", CreatedAt: time.Now().UTC()}), nil
}

// SetEventAssociation is not yet implemented in Phase 1
func (a *GeneratedEventsAdapter) SetEventAssociation(ctx context.Context, eventId string, req generated.SetAssociationRequest) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNotImplemented, generated.ErrorResponse{Message: "not implemented", CreatedAt: time.Now().UTC()}), nil
}

// RemoveEventAssociation is not yet implemented in Phase 1
func (a *GeneratedEventsAdapter) RemoveEventAssociation(ctx context.Context, eventId string) (generated.ImplResponse, error) {
	return generated.Response(http.StatusNotImplemented, generated.ErrorResponse{Message: "not implemented", CreatedAt: time.Now().UTC()}), nil
}

func ptr[T any](v T) *T { return &v }

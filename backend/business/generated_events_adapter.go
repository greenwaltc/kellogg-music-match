package business

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/jackc/pgx/v5/pgtype"
)

// GeneratedEventsAdapter adapts EventSearchService to the generated.EventsAPIServicer interface
// AssociatedEventsQuerier captures the sqlc method we need; enables fakes in tests.
type AssociatedEventsQuerier interface {
	GetAssociatedEventsPaged(ctx context.Context, arg sqlc.GetAssociatedEventsPagedParams) ([]sqlc.GetAssociatedEventsPagedRow, error)
	GetEventByID(ctx context.Context, id uuid.UUID) (sqlc.Event, error)
	GetEventBySourceExternal(ctx context.Context, arg sqlc.GetEventBySourceExternalParams) (sqlc.Event, error)
	InsertEvent(ctx context.Context, arg sqlc.InsertEventParams) (sqlc.Event, error)
	UpsertUserEventAssociation(ctx context.Context, arg sqlc.UpsertUserEventAssociationParams) error
	DeleteUserEventAssociation(ctx context.Context, arg sqlc.DeleteUserEventAssociationParams) error
	DeleteEventIfNoAssociations(ctx context.Context, eventID uuid.UUID) error
}

type GeneratedEventsAdapter struct {
	svc *EventSearchService
	cfg *config.Config
	db  AssociatedEventsQuerier
}

func NewGeneratedEventsAdapter(svc *EventSearchService, cfg *config.Config) *GeneratedEventsAdapter {
	// Best-effort: attach queries if we can get a pool from a repository in main; for now keep nil-safe.
	return &GeneratedEventsAdapter{svc: svc, cfg: cfg}
}

// NewGeneratedEventsAdapterWithQuerier allows injecting a DB querier (sqlc or fake)
func NewGeneratedEventsAdapterWithQuerier(svc *EventSearchService, cfg *config.Config, db AssociatedEventsQuerier) *GeneratedEventsAdapter {
	return &GeneratedEventsAdapter{svc: svc, cfg: cfg, db: db}
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
	// Public regardless of on-demand flag; this is local-only.
	// We require a DB handle. If not available, return empty page.
	if a.db == nil {
		empty := generated.EventsPage{Page: page, Size: size, Total: 0, Items: []generated.Event{}}
		return generated.Response(http.StatusOK, empty), nil
	}
	// Resolve current user (for myStatus) using a duck-typed interface stored under the legacy "user" key.
	type userIDGetter interface{ GetUserID() string }
	var myUserID *uuid.UUID
	if v := ctx.Value("user"); v != nil {
		if u, ok := v.(userIDGetter); ok {
			if id := u.GetUserID(); id != "" {
				if uid, err := uuid.Parse(id); err == nil {
					myUserID = &uid
				}
			}
		}
	}
	// sqlc args
	var startArg, endArg pgtype.Timestamptz
	if !startDateTime.IsZero() {
		startArg = pgtype.Timestamptz{Time: startDateTime, Valid: true}
	}
	if !endDateTime.IsZero() {
		endArg = pgtype.Timestamptz{Time: endDateTime, Valid: true}
	}
	seg := segmentName
	cty := city
	off := page * size
	lim := size
	uid := uuid.Nil
	if myUserID != nil {
		uid = *myUserID
	}
	rows, err := a.db.GetAssociatedEventsPaged(ctx, sqlc.GetAssociatedEventsPagedParams{
		MyUserID:    uid,
		StartFrom:   startArg,
		EndTo:       endArg,
		SegmentName: seg,
		City:        cty,
		OffSet:      off,
		Lim:         lim,
	})
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: err.Error(), CreatedAt: time.Now().UTC()}), nil
	}
	items := make([]generated.Event, 0, len(rows))
	var total int64 = 0
	for _, r := range rows {
		total = r.TotalCount
		// counts as int32
		interested := toInt32(r.InterestedCount)
		going := toInt32(r.GoingCount)
		lfg := toInt32(r.LfgCount)
		// myStatus optional
		var my *generated.EventInterestType
		if s, ok := r.MyStatus.(string); ok && s != "" {
			st, _ := generated.NewEventInterestTypeFromValue(s)
			my = &st
		}
		// Nullable fields
		venueName := valText(r.Venue, "Unknown Venue")
		cityName := valText(r.City, "")
		stateName := valText(r.State, "")
		country := valText(r.Country, "US")
		start := time.Now().UTC()
		if r.StartUtc.Valid {
			start = r.StartUtc.Time
		}
		url := valText(r.Url, "")

		ev := generated.Event{
			Id:         ptr(r.ID.String()),
			ExternalId: r.ExternalID,
			Source:     r.Source,
			Name:       r.Name,
			StartUtc:   start,
			Url:        url,
			Venue:      generated.Venue{Name: venueName},
			Location:   generated.EventLocation{City: cityName, State: stateName, Country: country},
			Association: generated.EventAssociation{
				MyStatus:        my,
				InterestedCount: interested,
				GoingCount:      going,
				LfgCount:        lfg,
			},
		}
		items = append(items, ev)
	}
	pageResp := generated.EventsPage{Page: page, Size: size, Total: int32(total), Items: items}
	return generated.Response(http.StatusOK, pageResp), nil
}

// SetEventAssociation is not yet implemented in Phase 1
func (a *GeneratedEventsAdapter) SetEventAssociation(ctx context.Context, eventId string, req generated.SetAssociationRequest) (generated.ImplResponse, error) {
	// Resolve user id
	type userIDGetter interface{ GetUserID() string }
	var myUser uuid.UUID
	if v := ctx.Value("user"); v != nil {
		if u, ok := v.(userIDGetter); ok {
			if id := u.GetUserID(); id != "" {
				if uid, err := uuid.Parse(id); err == nil {
					myUser = uid
				}
			}
		}
	}
	if myUser == uuid.Nil {
		return generated.Response(http.StatusUnauthorized, generated.ErrorResponse{Message: "unauthorized", CreatedAt: time.Now().UTC()}), nil
	}
	if a.db == nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: "database unavailable", CreatedAt: time.Now().UTC()}), nil
	}
	// Determine if eventId is internal UUID or external id
	var ev sqlc.Event
	var evID uuid.UUID
	if uid, err := uuid.Parse(eventId); err == nil {
		evID = uid
		if e, err := a.db.GetEventByID(ctx, uid); err == nil {
			ev = e
		}
	}
	if ev.ID == uuid.Nil {
		if e, err := a.db.GetEventBySourceExternal(ctx, sqlc.GetEventBySourceExternalParams{Source: "ticketmaster", ExternalID: eventId}); err == nil {
			ev = e
			evID = e.ID
		}
	}
	if ev.ID == uuid.Nil {
		// Create minimal event row on first association
		now := time.Now().UTC()
		raw, _ := json.Marshal(map[string]any{"externalId": eventId, "source": "ticketmaster"})
		newEv, err := a.db.InsertEvent(ctx, sqlc.InsertEventParams{
			Source:             ptr("ticketmaster"),
			ExternalID:         eventId,
			Name:               "",
			Venue:              pgtype.Text{},
			City:               pgtype.Text{},
			State:              pgtype.Text{},
			Country:            pgtype.Text{String: "US", Valid: true},
			StartUtc:           pgtype.Timestamptz{Time: now, Valid: true},
			Url:                pgtype.Text{},
			RawJson:            raw,
			SegmentName:        pgtype.Text{},
			ClassificationName: pgtype.Text{},
		})
		if err != nil {
			return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: err.Error(), CreatedAt: time.Now().UTC()}), nil
		}
		ev = newEv
		evID = newEv.ID
	}
	// Upsert association
	if err := a.db.UpsertUserEventAssociation(ctx, sqlc.UpsertUserEventAssociationParams{UserID: myUser, EventID: evID, Status: string(req.Status)}); err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: err.Error(), CreatedAt: time.Now().UTC()}), nil
	}
	return generated.Response(http.StatusNoContent, nil), nil
}

// RemoveEventAssociation is not yet implemented in Phase 1
func (a *GeneratedEventsAdapter) RemoveEventAssociation(ctx context.Context, eventId string) (generated.ImplResponse, error) {
	// Resolve user id
	type userIDGetter interface{ GetUserID() string }
	var myUser uuid.UUID
	if v := ctx.Value("user"); v != nil {
		if u, ok := v.(userIDGetter); ok {
			if id := u.GetUserID(); id != "" {
				if uid, err := uuid.Parse(id); err == nil {
					myUser = uid
				}
			}
		}
	}
	if myUser == uuid.Nil {
		return generated.Response(http.StatusUnauthorized, generated.ErrorResponse{Message: "unauthorized", CreatedAt: time.Now().UTC()}), nil
	}
	if a.db == nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: "database unavailable", CreatedAt: time.Now().UTC()}), nil
	}
	// Resolve event id (internal or external)
	var evID uuid.UUID
	if uid, err := uuid.Parse(eventId); err == nil {
		evID = uid
	} else {
		if e, err := a.db.GetEventBySourceExternal(ctx, sqlc.GetEventBySourceExternalParams{Source: "ticketmaster", ExternalID: eventId}); err == nil {
			evID = e.ID
		}
	}
	if evID == uuid.Nil {
		return generated.Response(http.StatusNotFound, generated.ErrorResponse{Message: "event not found", CreatedAt: time.Now().UTC()}), nil
	}
	if err := a.db.DeleteUserEventAssociation(ctx, sqlc.DeleteUserEventAssociationParams{UserID: myUser, EventID: evID}); err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: err.Error(), CreatedAt: time.Now().UTC()}), nil
	}
	_ = a.db.DeleteEventIfNoAssociations(ctx, evID)
	return generated.Response(http.StatusNoContent, nil), nil
}

func ptr[T any](v T) *T { return &v }

// helpers
func toInt32(v any) int32 {
	switch t := v.(type) {
	case int64:
		return int32(t)
	case int32:
		return t
	case int:
		return int32(t)
	case float64:
		return int32(t)
	default:
		return 0
	}
}
func valText(t pgtype.Text, def string) string {
	if t.Valid {
		return t.String
	}
	return def
}

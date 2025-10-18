package business_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	business "github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// userCtxLifecycle satisfies GetUserID for context user extraction
type userCtxLifecycle struct{ id string }

func (u userCtxLifecycle) GetUserID() string { return u.id }

type lifecycleFakeQuerier struct {
	// Inputs captured
	upsertArgs *sqlc.UpsertUserEventAssociationParams
	insertArgs *sqlc.InsertEventParams
	deleteArgs *sqlc.DeleteUserEventAssociationParams
	deleteEvID *uuid.UUID

	// Pre-configured behavior
	existingEventByID       *sqlc.Event
	existingEventByExternal *sqlc.Event
}

func (f *lifecycleFakeQuerier) GetAssociatedEventsPaged(ctx context.Context, arg sqlc.GetAssociatedEventsPagedParams) ([]sqlc.GetAssociatedEventsPagedRow, error) {
	return nil, nil
}
func (f *lifecycleFakeQuerier) GetEventByID(ctx context.Context, id uuid.UUID) (sqlc.Event, error) {
	if f.existingEventByID != nil && f.existingEventByID.ID == id {
		return *f.existingEventByID, nil
	}
	return sqlc.Event{}, errors.New("no rows")
}
func (f *lifecycleFakeQuerier) GetEventBySourceExternal(ctx context.Context, arg sqlc.GetEventBySourceExternalParams) (sqlc.Event, error) {
	if f.existingEventByExternal != nil && f.existingEventByExternal.ExternalID == arg.ExternalID && f.existingEventByExternal.Source == arg.Source {
		return *f.existingEventByExternal, nil
	}
	return sqlc.Event{}, errors.New("no rows")
}
func (f *lifecycleFakeQuerier) InsertEvent(ctx context.Context, arg sqlc.InsertEventParams) (sqlc.Event, error) {
	f.insertArgs = &arg
	// Return a constructed event row
	ev := sqlc.Event{
		ID:                 uuid.New(),
		Source:             derefInterfaceString(arg.Source),
		ExternalID:         arg.ExternalID,
		Name:               arg.Name,
		Venue:              arg.Venue,
		City:               arg.City,
		State:              arg.State,
		Country:            arg.Country,
		StartUtc:           arg.StartUtc,
		Url:                arg.Url,
		RawJson:            arg.RawJson,
		SegmentName:        arg.SegmentName,
		ClassificationName: arg.ClassificationName,
		CreatedAt:          pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:          pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	return ev, nil
}
func (f *lifecycleFakeQuerier) UpsertUserEventAssociation(ctx context.Context, arg sqlc.UpsertUserEventAssociationParams) error {
	cp := arg
	f.upsertArgs = &cp
	return nil
}
func (f *lifecycleFakeQuerier) DeleteUserEventAssociation(ctx context.Context, arg sqlc.DeleteUserEventAssociationParams) error {
	cp := arg
	f.deleteArgs = &cp
	return nil
}
func (f *lifecycleFakeQuerier) DeleteEventIfNoAssociations(ctx context.Context, eventID uuid.UUID) error {
	cp := eventID
	f.deleteEvID = &cp
	return nil
}

func derefInterfaceString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case *string:
		if t == nil {
			return ""
		}
		return *t
	default:
		return ""
	}
}

func TestSetEventAssociation_InsertsThenUpserts(t *testing.T) {
	cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{OnDemand: true}}
	fq := &lifecycleFakeQuerier{}
	adapter := business.NewGeneratedEventsAdapterWithQuerier(business.NewEventSearchService(nil), cfg, fq)

	// Call with external id where no event exists yet
	ctx := context.WithValue(context.Background(), "user", userCtxLifecycle{id: uuid.New().String()})
	req := generated.SetAssociationRequest{Status: generated.GOING}
	resp, err := adapter.SetEventAssociation(ctx, "TM-EXT-1", req)
	require.NoError(t, err)
	require.Equal(t, 204, resp.Code)

	// Verify insert then upsert captured
	require.NotNil(t, fq.insertArgs)
	require.Equal(t, "ticketmaster", derefInterfaceString(fq.insertArgs.Source))
	require.Equal(t, "TM-EXT-1", fq.insertArgs.ExternalID)
	require.NotNil(t, fq.upsertArgs)
	require.Equal(t, string(generated.GOING), fq.upsertArgs.Status)
	require.NotEqual(t, uuid.Nil, fq.upsertArgs.EventID)
	require.NotEqual(t, uuid.Nil, fq.upsertArgs.UserID)
}

func TestSetEventAssociation_UsesInternalUUID(t *testing.T) {
	cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{OnDemand: true}}
	fq := &lifecycleFakeQuerier{}

	// Pre-create an event accessible by ID
	evID := uuid.New()
	now := time.Now().UTC()
	fq.existingEventByID = &sqlc.Event{
		ID:         evID,
		Source:     "ticketmaster",
		ExternalID: "EXT-2",
		StartUtc:   pgtype.Timestamptz{Time: now, Valid: true},
		Country:    pgtype.Text{String: "US", Valid: true},
		CreatedAt:  pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:  pgtype.Timestamptz{Time: now, Valid: true},
	}

	adapter := business.NewGeneratedEventsAdapterWithQuerier(business.NewEventSearchService(nil), cfg, fq)
	ctx := context.WithValue(context.Background(), "user", userCtxLifecycle{id: uuid.New().String()})
	_, err := adapter.SetEventAssociation(ctx, evID.String(), generated.SetAssociationRequest{Status: generated.INTERESTED})
	require.NoError(t, err)
	require.NotNil(t, fq.upsertArgs)
	require.Equal(t, evID, fq.upsertArgs.EventID)
	// Should not have inserted
	require.Nil(t, fq.insertArgs)
}

func TestRemoveEventAssociation_ByExternalId(t *testing.T) {
	cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{OnDemand: true}}
	fq := &lifecycleFakeQuerier{}
	// Have an existing event by external id
	evID := uuid.New()
	now := time.Now().UTC()
	fq.existingEventByExternal = &sqlc.Event{ID: evID, Source: "ticketmaster", ExternalID: "EXT-3", StartUtc: pgtype.Timestamptz{Time: now, Valid: true}, Country: pgtype.Text{String: "US", Valid: true}, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true}}

	adapter := business.NewGeneratedEventsAdapterWithQuerier(business.NewEventSearchService(nil), cfg, fq)
	ctx := context.WithValue(context.Background(), "user", userCtxLifecycle{id: uuid.New().String()})
	resp, err := adapter.RemoveEventAssociation(ctx, "EXT-3")
	require.NoError(t, err)
	require.Equal(t, 204, resp.Code)
	require.NotNil(t, fq.deleteArgs)
	require.Equal(t, evID, fq.deleteArgs.EventID)
	require.NotNil(t, fq.deleteEvID)
	require.Equal(t, evID, *fq.deleteEvID)
}

func TestRemoveEventAssociation_NotFound(t *testing.T) {
	cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{OnDemand: true}}
	fq := &lifecycleFakeQuerier{}
	adapter := business.NewGeneratedEventsAdapterWithQuerier(business.NewEventSearchService(nil), cfg, fq)
	ctx := context.WithValue(context.Background(), "user", userCtxLifecycle{id: uuid.New().String()})
	resp, err := adapter.RemoveEventAssociation(ctx, "DOES-NOT-EXIST")
	require.NoError(t, err)
	require.Equal(t, 404, resp.Code)
}

func TestLifecycle_Unauthorized(t *testing.T) {
	cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{OnDemand: true}}
	fq := &lifecycleFakeQuerier{}
	adapter := business.NewGeneratedEventsAdapterWithQuerier(business.NewEventSearchService(nil), cfg, fq)
	// No user in context
	_, err := adapter.SetEventAssociation(context.Background(), "X", generated.SetAssociationRequest{Status: generated.LOOKING_FOR_GROUP})
	require.NoError(t, err)
	// Remove should also be unauthorized without user
	resp, _ := adapter.RemoveEventAssociation(context.Background(), "X")
	require.Equal(t, 401, resp.Code)
}

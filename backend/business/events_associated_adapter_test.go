package business_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	business "github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeQuerier implements the minimal method for AssociatedEventsQuerier
type fakeQuerier struct {
	rows   []sqlc.GetAssociatedEventsPagedRow
	params *sqlc.GetAssociatedEventsPagedParams
}

func (f *fakeQuerier) GetAssociatedEventsPaged(ctx context.Context, arg sqlc.GetAssociatedEventsPagedParams) ([]sqlc.GetAssociatedEventsPagedRow, error) {
	// capture params for assertions
	cp := arg
	f.params = &cp
	return f.rows, nil
}

// Stubs to satisfy the full AssociatedEventsQuerier interface; unused in these tests
func (f *fakeQuerier) GetEventByID(ctx context.Context, id uuid.UUID) (sqlc.Event, error) {
	return sqlc.Event{}, nil
}
func (f *fakeQuerier) GetEventBySourceExternal(ctx context.Context, arg sqlc.GetEventBySourceExternalParams) (sqlc.Event, error) {
	return sqlc.Event{}, nil
}
func (f *fakeQuerier) InsertEvent(ctx context.Context, arg sqlc.InsertEventParams) (sqlc.Event, error) {
	return sqlc.Event{}, nil
}
func (f *fakeQuerier) UpsertUserEventAssociation(ctx context.Context, arg sqlc.UpsertUserEventAssociationParams) error {
	return nil
}
func (f *fakeQuerier) DeleteUserEventAssociation(ctx context.Context, arg sqlc.DeleteUserEventAssociationParams) error {
	return nil
}
func (f *fakeQuerier) DeleteEventIfNoAssociations(ctx context.Context, eventID uuid.UUID) error {
	return nil
}

// userCtx is a minimal type that satisfies the adapter's userIDGetter via method name
type userCtx struct{ id string }

func (u userCtx) GetUserID() string { return u.id }

var _ = ginkgo.Describe("Associated Events mapping", func() {
	ginkgo.It("returns empty page when DB has no rows", func() {
		cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{OnDemand: true}}
		// service not used by GetAssociatedEvents; pass a dummy
		adapter := business.NewGeneratedEventsAdapterWithQuerier(business.NewEventSearchService(nil), cfg, &fakeQuerier{rows: []sqlc.GetAssociatedEventsPagedRow{}})

		resp, err := adapter.GetAssociatedEvents(context.Background(), time.Time{}, time.Time{}, "", "", 10, 0)
		Expect(err).To(BeNil())
		Expect(resp.Code).To(Equal(200))
		page, ok := resp.Body.(generated.EventsPage)
		Expect(ok).To(BeTrue())
		Expect(page.Total).To(Equal(int32(0)))
		Expect(page.Items).To(HaveLen(0))
	})

	ginkgo.It("maps rows with counts and myStatus and applies pagination params", func() {
		cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{OnDemand: true}}
		fq := &fakeQuerier{}
		adapter := business.NewGeneratedEventsAdapterWithQuerier(business.NewEventSearchService(nil), cfg, fq)

		// prepare two rows
		id1 := uuid.New()
		id2 := uuid.New()
		now := time.Now().UTC()
		fq.rows = []sqlc.GetAssociatedEventsPagedRow{
			{
				ID:                 id1,
				Source:             "ticketmaster",
				ExternalID:         "E1",
				Name:               "Show 1",
				Venue:              pgtype.Text{String: "Venue 1", Valid: true},
				City:               pgtype.Text{String: "Chicago", Valid: true},
				State:              pgtype.Text{String: "IL", Valid: true},
				Country:            pgtype.Text{String: "US", Valid: true},
				StartUtc:           pgtype.Timestamptz{Time: now, Valid: true},
				Url:                pgtype.Text{String: "https://e1", Valid: true},
				SegmentName:        pgtype.Text{String: "Music", Valid: true},
				ClassificationName: pgtype.Text{String: "Pop", Valid: true},
				InterestedCount:    int64(3),
				GoingCount:         int64(1),
				LfgCount:           int64(2),
				MyStatus:           "INTERESTED",
				TotalCount:         2,
			},
			{
				ID:                 id2,
				Source:             "ticketmaster",
				ExternalID:         "E2",
				Name:               "Show 2",
				Venue:              pgtype.Text{String: "Venue 2", Valid: true},
				City:               pgtype.Text{String: "Chicago", Valid: true},
				State:              pgtype.Text{String: "IL", Valid: true},
				Country:            pgtype.Text{String: "US", Valid: true},
				StartUtc:           pgtype.Timestamptz{Time: now.Add(24 * time.Hour), Valid: true},
				Url:                pgtype.Text{String: "https://e2", Valid: true},
				SegmentName:        pgtype.Text{String: "Music", Valid: true},
				ClassificationName: pgtype.Text{String: "Rock", Valid: true},
				InterestedCount:    int64(0),
				GoingCount:         int64(0),
				LfgCount:           int64(1),
				MyStatus:           nil,
				TotalCount:         2,
			},
		}

		// context with user id to enable myStatus mapping
		uid := uuid.New().String()
		ctx := context.WithValue(context.Background(), "user", userCtx{id: uid})

		// size=10, page=2 -> expect OffSet=20 in captured params
		resp, err := adapter.GetAssociatedEvents(ctx, time.Time{}, time.Time{}, "Music", "Chicago", 10, 2)
		Expect(err).To(BeNil())
		Expect(resp.Code).To(Equal(200))
		page, ok := resp.Body.(generated.EventsPage)
		Expect(ok).To(BeTrue())
		Expect(page.Total).To(Equal(int32(2)))
		Expect(page.Items).To(HaveLen(2))

		// first event assertions
		e1 := page.Items[0]
		Expect(*e1.Id).To(Equal(id1.String()))
		Expect(e1.ExternalId).To(Equal("E1"))
		Expect(e1.Source).To(Equal("ticketmaster"))
		Expect(e1.Name).To(Equal("Show 1"))
		Expect(e1.Venue.Name).To(Equal("Venue 1"))
		Expect(e1.Location.City).To(Equal("Chicago"))
		Expect(e1.Location.State).To(Equal("IL"))
		Expect(e1.Location.Country).To(Equal("US"))
		Expect(e1.Url).To(Equal("https://e1"))
		Expect(e1.Association.InterestedCount).To(Equal(int32(3)))
		Expect(e1.Association.GoingCount).To(Equal(int32(1)))
		Expect(e1.Association.LfgCount).To(Equal(int32(2)))
		Expect(e1.Association.MyStatus).ToNot(BeNil())
		Expect(*e1.Association.MyStatus).To(Equal(generated.INTERESTED))

		// second event assertions
		e2 := page.Items[1]
		Expect(*e2.Id).To(Equal(id2.String()))
		Expect(e2.Association.MyStatus).To(BeNil())

		// pagination params were captured
		Expect(fq.params).ToNot(BeNil())
		Expect(fq.params.SegmentName).To(Equal("Music"))
		Expect(fq.params.City).To(Equal("Chicago"))
		Expect(fq.params.Lim).To(Equal(int32(10)))
		Expect(fq.params.OffSet).To(Equal(int32(20)))
		// user id parsed and passed
		Expect(fq.params.MyUserID.String()).To(Equal(uid))
	})
})

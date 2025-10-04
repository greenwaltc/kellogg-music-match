package business

import (
	"context"
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Focused tests for new ChicagoEvents filtering dimensions (date range, genres, interest statuses, anyInterest, combinations)
func TestConcertAPIService_ChicagoEventsFilters(t *testing.T) {
	ctx := context.Background()
	repo := concert.NewMockRepository()
	provider := &TestEventProvider{}
	cfg := &config.Config{}
	api := NewConcertAPIServiceWithRepository(provider, repo, cfg)

	baseDate := time.Now().Add(24 * time.Hour).Truncate(time.Hour)

	makeEvent := func(id, name string, d time.Time, genres []string) *concert.Event {
		return &concert.Event{
			ID:      id,
			Name:    name,
			Date:    d,
			Venue:   concert.Venue{ID: "v-" + id, Name: "Venue " + id, Address: concert.Address{City: "Chicago", Country: "US"}},
			Artists: []concert.Artist{{ID: "a-" + id, Name: name + " Artist", Genres: genres}},
			Genres:  genres,
			Status:  "onsale",
		}
	}

	ev1 := makeEvent("e1", "Alpha", baseDate, []string{"Rock", "Indie"})
	ev2 := makeEvent("e2", "Beta", baseDate.Add(48*time.Hour), []string{"Pop"})
	ev3 := makeEvent("e3", "Gamma", baseDate.Add(72*time.Hour), []string{"Jazz", "Funk"})

	// Add interest to ev2 (GOING) and ev3 (INTERESTED)
	require.NoError(t, repo.UpsertEvent(ctx, ev1))
	require.NoError(t, repo.UpsertEvent(ctx, ev2))
	require.NoError(t, repo.UpsertEvent(ctx, ev3))

	// Mark user interests in mock
	_ = repo.UpsertUserInterest(ctx, "user-1", "e2", "GOING")
	_ = repo.UpsertUserInterest(ctx, "user-2", "e3", "INTERESTED")

	t.Run("GenreFilterSingle", func(t *testing.T) {
		resp, err := api.GetChicagoEvents(ctx, "", 10, 0, time.Time{}, time.Time{}, "Rock", "", false)
		require.NoError(t, err)
		result := resp.Body.(generated.ChicagoEventsResult)
		require.Len(t, result.Events, 1)
		assert.Equal(t, "Alpha", result.Events[0].Name)
	})

	t.Run("GenreFilterMultipleAnyOverlap", func(t *testing.T) {
		resp, err := api.GetChicagoEvents(ctx, "", 10, 0, time.Time{}, time.Time{}, "Jazz,Pop", "", false)
		require.NoError(t, err)
		result := resp.Body.(generated.ChicagoEventsResult)
		names := []string{result.Events[0].Name, result.Events[1].Name}
		assert.ElementsMatch(t, []string{"Beta", "Gamma"}, names)
	})

	t.Run("DateRange", func(t *testing.T) {
		start := baseDate.Add(36 * time.Hour)
		end := baseDate.Add(80 * time.Hour)
		resp, err := api.GetChicagoEvents(ctx, "", 10, 0, start, end, "", "", false)
		require.NoError(t, err)
		result := resp.Body.(generated.ChicagoEventsResult)
		// Expect ev2, ev3 only
		require.Len(t, result.Events, 2)
		assert.Equal(t, "Beta", result.Events[0].Name)
		assert.Equal(t, "Gamma", result.Events[1].Name)
	})

	t.Run("InterestStatuses", func(t *testing.T) {
		resp, err := api.GetChicagoEvents(ctx, "", 10, 0, time.Time{}, time.Time{}, "", "GOING", false)
		require.NoError(t, err)
		result := resp.Body.(generated.ChicagoEventsResult)
		require.Len(t, result.Events, 1)
		assert.Equal(t, "Beta", result.Events[0].Name)
	})

	t.Run("AnyInterest", func(t *testing.T) {
		resp, err := api.GetChicagoEvents(ctx, "", 10, 0, time.Time{}, time.Time{}, "", "", true)
		require.NoError(t, err)
		result := resp.Body.(generated.ChicagoEventsResult)
		// ev2 and ev3 have interest
		require.Len(t, result.Events, 2)
		names := []string{result.Events[0].Name, result.Events[1].Name}
		assert.ElementsMatch(t, []string{"Beta", "Gamma"}, names)
	})

	t.Run("CombinedGenreAndInterest", func(t *testing.T) {
		resp, err := api.GetChicagoEvents(ctx, "", 10, 0, time.Time{}, time.Time{}, "Jazz,Indie", "INTERESTED", false)
		require.NoError(t, err)
		result := resp.Body.(generated.ChicagoEventsResult)
		// Should only return Gamma (INTERESTED + Jazz)
		require.Len(t, result.Events, 1)
		assert.Equal(t, "Gamma", result.Events[0].Name)
	})
}

package concert

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockRepository_GetChicagoEvents(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Create test events
	chicagoEvent1 := &Event{
		ID:   "chicago-1",
		Name: "Taylor Swift Concert",
		Date: time.Now().Add(24 * time.Hour),
		Venue: Venue{
			ID:   "venue-1",
			Name: "United Center",
			Address: Address{
				City:    "Chicago",
				State:   "IL",
				Country: "US",
			},
		},
		Artists: []Artist{
			{ID: "artist-1", Name: "Taylor Swift", Genres: []string{"Pop"}},
		},
		Status: "onsale",
	}

	chicagoEvent2 := &Event{
		ID:   "chicago-2",
		Name: "Metallica Show",
		Date: time.Now().Add(48 * time.Hour),
		Venue: Venue{
			ID:   "venue-2",
			Name: "Soldier Field",
			Address: Address{
				City:    "Chicago",
				State:   "IL",
				Country: "US",
			},
		},
		Artists: []Artist{
			{ID: "artist-2", Name: "Metallica", Genres: []string{"Metal"}},
		},
		Status: "onsale",
	}

	nyEvent := &Event{
		ID:   "ny-1",
		Name: "Broadway Show",
		Date: time.Now().Add(72 * time.Hour),
		Venue: Venue{
			ID:   "venue-3",
			Name: "Madison Square Garden",
			Address: Address{
				City:    "New York",
				State:   "NY",
				Country: "US",
			},
		},
		Artists: []Artist{
			{ID: "artist-3", Name: "Broadway Cast", Genres: []string{"Musical"}},
		},
		Status: "onsale",
	}

	// Add events to repository
	require.NoError(t, repo.UpsertEvent(ctx, chicagoEvent1))
	require.NoError(t, repo.UpsertEvent(ctx, chicagoEvent2))
	require.NoError(t, repo.UpsertEvent(ctx, nyEvent))

	t.Run("GetAllChicagoEvents", func(t *testing.T) {
		events, err := repo.GetChicagoEvents(ctx, nil, false, 10, 0)
		require.NoError(t, err)
		assert.Len(t, events, 2, "Should return only Chicago events")

		// Verify events are Chicago events
		for _, event := range events {
			assert.Equal(t, "Chicago", event.Venue.Address.City)
		}
	})

	t.Run("SearchByArtistName", func(t *testing.T) {
		artistName := "Taylor"
		events, err := repo.GetChicagoEvents(ctx, &artistName, false, 10, 0)
		require.NoError(t, err)
		assert.Len(t, events, 1, "Should return only Taylor Swift event")
		assert.Equal(t, "Taylor Swift Concert", events[0].Name)
	})

	t.Run("SearchByArtistNameCaseInsensitive", func(t *testing.T) {
		artistName := "metallica"
		events, err := repo.GetChicagoEvents(ctx, &artistName, false, 10, 0)
		require.NoError(t, err)
		assert.Len(t, events, 1, "Should return Metallica event with case insensitive search")
		assert.Equal(t, "Metallica Show", events[0].Name)
	})

	t.Run("PaginationLimit", func(t *testing.T) {
		events, err := repo.GetChicagoEvents(ctx, nil, false, 1, 0)
		require.NoError(t, err)
		assert.Len(t, events, 1, "Should respect limit parameter")
	})

	t.Run("PaginationOffset", func(t *testing.T) {
		events, err := repo.GetChicagoEvents(ctx, nil, false, 1, 1)
		require.NoError(t, err)
		assert.Len(t, events, 1, "Should respect offset parameter")
	})

	t.Run("NoMatchingArtist", func(t *testing.T) {
		artistName := "NonexistentArtist"
		events, err := repo.GetChicagoEvents(ctx, &artistName, false, 10, 0)
		require.NoError(t, err)
		assert.Len(t, events, 0, "Should return no events for nonexistent artist")
	})
}

func TestMockRepository_GetChicagoEventsCount(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Create test events
	chicagoEvent := &Event{
		ID:   "chicago-1",
		Name: "Chicago Event",
		Date: time.Now().Add(24 * time.Hour),
		Venue: Venue{
			Address: Address{City: "Chicago"},
		},
		Artists: []Artist{
			{Name: "Test Artist", Genres: []string{"Rock"}},
		},
	}

	nonChicagoEvent := &Event{
		ID:   "ny-1",
		Name: "NY Event",
		Date: time.Now().Add(24 * time.Hour),
		Venue: Venue{
			Address: Address{City: "New York"},
		},
		Artists: []Artist{
			{Name: "Other Artist", Genres: []string{"Pop"}},
		},
	}

	require.NoError(t, repo.UpsertEvent(ctx, chicagoEvent))
	require.NoError(t, repo.UpsertEvent(ctx, nonChicagoEvent))

	t.Run("CountAllChicagoEvents", func(t *testing.T) {
		count, err := repo.GetChicagoEventsCount(ctx, nil, false)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should count only Chicago events")
	})

	t.Run("CountWithArtistFilter", func(t *testing.T) {
		artistName := "Test"
		count, err := repo.GetChicagoEventsCount(ctx, &artistName, false)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should count Chicago events matching artist")
	})

	t.Run("CountWithNoMatchingArtist", func(t *testing.T) {
		artistName := "Nonexistent"
		count, err := repo.GetChicagoEventsCount(ctx, &artistName, false)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "Should return 0 for nonexistent artist")
	})
}

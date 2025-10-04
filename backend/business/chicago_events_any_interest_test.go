package business

import (
	"context"
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Focused test verifying anyInterest=true only returns events that have at least one user expressing interest.
func TestConcertService_GetChicagoEvents_AnyInterest(t *testing.T) {
	repo := concert.NewMockRepository()
	ctx := context.Background()

	baseTime := time.Now().Add(2 * time.Hour)
	// Event without interest
	e1 := &concert.Event{
		ID:      "evt-no-interest",
		Name:    "Silent Show",
		Date:    baseTime,
		Venue:   concert.Venue{ID: "v1", Name: "Venue 1", Address: concert.Address{City: "Chicago", Country: "US"}},
		Artists: []concert.Artist{{ID: "a1", Name: "Band A"}},
		Status:  "onsale",
	}
	// Event with interest
	e2 := &concert.Event{
		ID:      "evt-with-interest",
		Name:    "Popular Show",
		Date:    baseTime.Add(1 * time.Hour),
		Venue:   concert.Venue{ID: "v2", Name: "Venue 2", Address: concert.Address{City: "Chicago", Country: "US"}},
		Artists: []concert.Artist{{ID: "a2", Name: "Band B"}},
		Status:  "onsale",
	}

	require.NoError(t, repo.UpsertEvent(ctx, e1))
	require.NoError(t, repo.UpsertEvent(ctx, e2))
	// Add interest to second event
	require.NoError(t, repo.UpsertUserInterest(ctx, "user-123", e2.ID, "INTERESTED"))

	svc := NewConcertServiceWithRepository(&MockEventProvider{}, repo, &config.Config{})

	// anyInterest=false returns both
	eventsAll, countAll, err := svc.GetChicagoEvents(ctx, nil, false, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), countAll)
	require.Equal(t, 2, len(eventsAll))

	// anyInterest=true returns only the one with interest
	eventsFiltered, countFiltered, err := svc.GetChicagoEvents(ctx, nil, true, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, len(eventsFiltered))
	assert.Equal(t, int64(1), countFiltered)
	assert.Equal(t, "evt-with-interest", eventsFiltered[0].ID)
}

package concert

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockRepository_UserInterestLifecycle(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	ev := &Event{
		ID:      "evt-1",
		Name:    "Test Show",
		Date:    time.Now().Add(24 * time.Hour),
		Venue:   Venue{ID: "v1", Name: "Test Venue", Address: Address{City: "Chicago", Country: "US"}},
		Artists: []Artist{{ID: "a1", Name: "Band", Genres: []string{"Rock"}}},
		Status:  "onsale",
	}
	require.NoError(t, repo.UpsertEvent(ctx, ev))

	// Upsert interests
	require.NoError(t, repo.UpsertUserInterest(ctx, "user-1", "evt-1", "INTERESTED"))
	require.NoError(t, repo.UpsertUserInterest(ctx, "user-2", "evt-1", "GOING"))
	require.NoError(t, repo.UpsertUserInterest(ctx, "user-3", "evt-1", "LOOKING_FOR_GROUP"))

	events, err := repo.GetChicagoEvents(ctx, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	e := events[0]
	assert.ElementsMatch(t, []string{"user-1"}, e.InterestedUserIDs)
	assert.ElementsMatch(t, []string{"user-2"}, e.GoingUserIDs)
	assert.ElementsMatch(t, []string{"user-3"}, e.LookingForGroupUserIDs)

	// Update interest status
	require.NoError(t, repo.UpsertUserInterest(ctx, "user-1", "evt-1", "GOING"))
	events, err = repo.GetChicagoEvents(ctx, nil, 10, 0)
	require.NoError(t, err)
	e = events[0]
	assert.ElementsMatch(t, []string{}, e.InterestedUserIDs)
	assert.ElementsMatch(t, []string{"user-1", "user-2"}, e.GoingUserIDs)

	// Remove interest
	require.NoError(t, repo.RemoveUserInterest(ctx, "user-2", "evt-1"))
	events, err = repo.GetChicagoEvents(ctx, nil, 10, 0)
	require.NoError(t, err)
	e = events[0]
	assert.ElementsMatch(t, []string{}, e.InterestedUserIDs)
	assert.ElementsMatch(t, []string{"user-1"}, e.GoingUserIDs)
}

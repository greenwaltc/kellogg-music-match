package business
package business

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcertAPIService_GetChicagoEvents(t *testing.T) {
	// Create mock repository with test data
	mockRepo := concert.NewMockRepository()
	mockProvider := &TestEventProvider{}
	ctx := context.Background()

	// Create test events
	testEvents := []*concert.Event{
		{
			ID:   "chicago-1",
			Name: "Rock Concert",
			Date: time.Now().Add(24 * time.Hour),
			Venue: concert.Venue{
				ID:   "venue-1",
				Name: "Chicago Theater",
				Address: concert.Address{
					Street:  "175 N State St",
					City:    "Chicago",
					State:   "IL",
					Country: "US",
					Postal:  "60601",
				},
			},
			Artists: []concert.Artist{
				{ID: "artist-1", Name: "Rock Band", Genres: []string{"Rock"}},
			},
			Status:    "onsale",
			TicketURL: "https://example.com/tickets/1",
		},
		{
			ID:   "chicago-2",
			Name: "Pop Concert",
			Date: time.Now().Add(48 * time.Hour),
			Venue: concert.Venue{
				ID:   "venue-2",
				Name: "United Center",
				Address: concert.Address{
					Street:  "1901 W Madison St",
					City:    "Chicago",
					State:   "IL",
					Country: "US",
					Postal:  "60612",
				},
			},
			Artists: []concert.Artist{
				{ID: "artist-2", Name: "Pop Star", Genres: []string{"Pop"}},
			},
			Status:    "onsale",
			TicketURL: "https://example.com/tickets/2",
		},
	}

	// Add events to mock repository
	for _, event := range testEvents {
		require.NoError(t, mockRepo.UpsertEvent(ctx, event))
	}

	// Create API service
	cfg := &config.Config{}
	service := NewConcertAPIServiceWithRepository(mockProvider, mockRepo, cfg)

	t.Run("GetAllChicagoEvents", func(t *testing.T) {
		response, err := service.GetChicagoEvents(ctx, "", 10, 0)
		require.NoError(t, err)
		
		assert.Equal(t, http.StatusOK, response.Code)
		
		result, ok := response.Body.(generated.ChicagoEventsResult)
		require.True(t, ok, "Response should be ChicagoEventsResult")
		
		assert.Len(t, result.Events, 2, "Should return both events")
		assert.Equal(t, int32(2), result.TotalCount, "Should have correct total count")
		assert.False(t, result.HasMore, "Should not have more events")
		
		// Verify events are sorted by date (earliest first)
		assert.Equal(t, "Rock Concert", result.Events[0].Name)
		assert.Equal(t, "Pop Concert", result.Events[1].Name)
	})

	t.Run("SearchByArtistName", func(t *testing.T) {
		response, err := service.GetChicagoEvents(ctx, "Rock", 10, 0)
		require.NoError(t, err)
		
		assert.Equal(t, http.StatusOK, response.Code)
		
		result, ok := response.Body.(generated.ChicagoEventsResult)
		require.True(t, ok, "Response should be ChicagoEventsResult")
		
		assert.Len(t, result.Events, 1, "Should return only Rock Band event")
		assert.Equal(t, "Rock Concert", result.Events[0].Name)
		assert.Equal(t, int32(1), result.TotalCount, "Should have correct total count")
	})

	t.Run("PaginationWithLimit", func(t *testing.T) {
		response, err := service.GetChicagoEvents(ctx, "", 1, 0)
		require.NoError(t, err)
		
		result, ok := response.Body.(generated.ChicagoEventsResult)
		require.True(t, ok, "Response should be ChicagoEventsResult")
		
		assert.Len(t, result.Events, 1, "Should return only 1 event due to limit")
		assert.True(t, result.HasMore, "Should indicate more events available")
		assert.Equal(t, int32(2), result.TotalCount, "Should have correct total count")
	})

	t.Run("PaginationWithOffset", func(t *testing.T) {
		response, err := service.GetChicagoEvents(ctx, "", 1, 1)
		require.NoError(t, err)
		
		result, ok := response.Body.(generated.ChicagoEventsResult)
		require.True(t, ok, "Response should be ChicagoEventsResult")
		
		assert.Len(t, result.Events, 1, "Should return 1 event")
		assert.Equal(t, "Pop Concert", result.Events[0].Name, "Should return second event due to offset")
		assert.False(t, result.HasMore, "Should not have more events")
	})

	t.Run("NoResults", func(t *testing.T) {
		response, err := service.GetChicagoEvents(ctx, "NonexistentArtist", 10, 0)
		require.NoError(t, err)
		
		result, ok := response.Body.(generated.ChicagoEventsResult)
		require.True(t, ok, "Response should be ChicagoEventsResult")
		
		assert.Len(t, result.Events, 0, "Should return no events")
		assert.Equal(t, int32(0), result.TotalCount, "Should have zero total count")
		assert.False(t, result.HasMore, "Should not have more events")
	})

	t.Run("EventDataIntegrity", func(t *testing.T) {
		response, err := service.GetChicagoEvents(ctx, "", 1, 0)
		require.NoError(t, err)
		
		result, ok := response.Body.(generated.ChicagoEventsResult)
		require.True(t, ok, "Response should be ChicagoEventsResult")
		
		event := result.Events[0]
		
		// Verify complete event data
		assert.Equal(t, "chicago-1", event.Id)
		assert.Equal(t, "Rock Concert", event.Name)
		assert.NotEmpty(t, event.Date, "Date should be set")
		
		// Verify venue data
		assert.Equal(t, "Chicago Theater", event.Venue.Name)
		assert.Equal(t, "175 N State St", event.Venue.Address.Street)
		assert.Equal(t, "Chicago", event.Venue.Address.City)
		assert.Equal(t, "IL", event.Venue.Address.State)
		assert.Equal(t, "US", event.Venue.Address.Country)
		
		// Verify artist data
		assert.Len(t, event.Artists, 1)
		assert.Equal(t, "Rock Band", event.Artists[0].Name)
		assert.Equal(t, []string{"Rock"}, event.Artists[0].Genres)
		
		// Verify ticket URL
		assert.Equal(t, "https://example.com/tickets/1", event.TicketUrl)
	})
}

// TestEventProvider is a mock implementation for testing
type TestEventProvider struct{}

func (t *TestEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	return &concert.SearchResult{}, nil
}

func (t *TestEventProvider) GetEventByID(ctx context.Context, eventID string) (*concert.Event, error) {
	return nil, concert.ErrEventNotFound
}

func (t *TestEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	return &concert.SearchResult{}, nil
}

func (t *TestEventProvider) GetProviderName() string {
	return "Test"
}

func (t *TestEventProvider) IsHealthy(ctx context.Context) error {
	return nil
}
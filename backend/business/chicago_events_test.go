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

func TestConcertService_GetChicagoEvents(t *testing.T) {
	// Create mock repository with test data
	mockRepo := concert.NewMockRepository()
	ctx := context.Background()

	// Create test events
	testEvent := &concert.Event{
		ID:   "test-1",
		Name: "Test Concert",
		Date: time.Now().Add(24 * time.Hour),
		Venue: concert.Venue{
			ID:   "venue-1",
			Name: "Test Venue",
			Address: concert.Address{
				City:    "Chicago",
				State:   "IL",
				Country: "US",
			},
		},
		Artists: []concert.Artist{
			{ID: "artist-1", Name: "Test Artist", Genres: []string{"Rock"}},
		},
		Status: "onsale",
	}

	require.NoError(t, mockRepo.UpsertEvent(ctx, testEvent))

	// Create service with mock repository
	cfg := &config.Config{}
	mockProvider := &MockEventProvider{}
	service := NewConcertServiceWithRepository(mockProvider, mockRepo, cfg)

	t.Run("GetChicagoEventsSuccess", func(t *testing.T) {
		events, count, err := service.GetChicagoEvents(ctx, nil, false, 10, 0)
		require.NoError(t, err)
		assert.Len(t, events, 1, "Should return test event")
		assert.Equal(t, int64(1), count, "Should return correct count")
		assert.Equal(t, "Test Concert", events[0].Name)
	})

	t.Run("GetChicagoEventsWithArtistFilter", func(t *testing.T) {
		artistName := "Test"
		events, count, err := service.GetChicagoEvents(ctx, &artistName, false, 10, 0)
		require.NoError(t, err)
		assert.Len(t, events, 1, "Should return filtered event")
		assert.Equal(t, int64(1), count, "Should return correct count")
	})

	t.Run("GetChicagoEventsNoRepository", func(t *testing.T) {
		serviceWithoutRepo := NewConcertService(cfg)
		_, _, err := serviceWithoutRepo.GetChicagoEvents(ctx, nil, false, 10, 0)
		assert.Error(t, err, "Should return error when repository not available")
		assert.Contains(t, err.Error(), "repository not available")
	})
}

// MockEventProvider is a simple mock implementation for testing
type MockEventProvider struct{}

func (m *MockEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	return &concert.SearchResult{
		Events:      []concert.Event{},
		TotalCount:  0,
		CurrentPage: 0,
		TotalPages:  0,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) GetEventByID(ctx context.Context, eventID string) (*concert.Event, error) {
	return nil, concert.ErrEventNotFound
}

func (m *MockEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	return &concert.SearchResult{
		Events:      []concert.Event{},
		TotalCount:  0,
		CurrentPage: 0,
		TotalPages:  0,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) GetProviderName() string {
	return "Mock"
}

func (m *MockEventProvider) IsHealthy(ctx context.Context) error {
	return nil
}

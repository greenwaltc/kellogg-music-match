package business

import (

	"context"

	"testing"import (

	"time"

	"context"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"

	"github.com/greenwaltc/kellogg-music-match/backend/config"import (import (

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"

)

)	"context"	"context"

func TestConcertService_GetChicagoEvents(t *testing.T) {

	// Create mock repository with test data

	mockRepo := concert.NewMockRepository()

	ctx := context.Background()// MockEventProvider implements concert.EventProvider for testing	"fmt"



	// Create test eventstype MockEventProvider struct {

	testEvent := &concert.Event{

		ID:   "test-1",	events []concert.Event	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"

		Name: "Test Concert",

		Date: time.Now().Add(24 * time.Hour),	errors map[string]error

		Venue: concert.Venue{

			ID:   "venue-1",})	"github.com/greenwaltc/kellogg-music-match/backend/business"

			Name: "Test Venue",

			Address: concert.Address{

				City:    "Chicago",

				State:   "IL",func (m *MockEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"

				Country: "US",

			},	if err, exists := m.errors["search"]; exists {

		},

		Artists: []concert.Artist{		return nil, err// MockEventProvider implements concert.EventProvider for testing	"github.com/greenwaltc/kellogg-music-match/backend/config"

			{ID: "artist-1", Name: "Test Artist", Genres: []string{"Rock"}},

		},	}

		Status: "onsale",

	}	type MockEventProvider struct {	. "github.com/onsi/ginkgo/v2"



	require.NoError(t, mockRepo.UpsertEvent(ctx, testEvent))	return &concert.SearchResult{



	// Create service with mock repository		Events:      m.events,	events []concert.Event	. "github.com/onsi/gomega"

	cfg := &config.Config{}

	mockProvider := &MockEventProvider{}		TotalCount:  len(m.events),

	service := NewConcertServiceWithRepository(mockProvider, mockRepo, cfg)

		CurrentPage: 0,	errors map[string]error)

	t.Run("GetChicagoEventsSuccess", func(t *testing.T) {

		events, count, err := service.GetChicagoEvents(ctx, nil, 10, 0)		TotalPages:  1,

		require.NoError(t, err)

		assert.Len(t, events, 1, "Should return test event")		HasMore:     false,}

		assert.Equal(t, int64(1), count, "Should return correct count")

		assert.Equal(t, "Test Concert", events[0].Name)	}, nil

	})

}var _ = Describe("Concert Service", func() {

	t.Run("GetChicagoEventsWithArtistFilter", func(t *testing.T) {

		artistName := "Test"

		events, count, err := service.GetChicagoEvents(ctx, &artistName, 10, 0)

		require.NoError(t, err)func (m *MockEventProvider) GetEventByID(ctx context.Context, id string) (*concert.Event, error) {func (m *MockEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {	var (

		assert.Len(t, events, 1, "Should return filtered event")

		assert.Equal(t, int64(1), count, "Should return correct count")	if err, exists := m.errors["getById"]; exists {

	})

		return nil, err	if err, exists := m.errors["search"]; exists {		service *business.ConcertService

	t.Run("GetChicagoEventsNoRepository", func(t *testing.T) {

		serviceWithoutRepo := NewConcertService(cfg)	}

		_, _, err := serviceWithoutRepo.GetChicagoEvents(ctx, nil, 10, 0)

		assert.Error(t, err, "Should return error when repository not available")			return nil, err		cfg     *config.Config

		assert.Contains(t, err.Error(), "repository not available")

	})	for _, event := range m.events {

}

		if event.ID == id {	}	)

// MockEventProvider is a simple mock implementation for testing

type MockEventProvider struct{}			return &event, nil



func (m *MockEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {		}	

	return &concert.SearchResult{

		Events:     []concert.Event{},	}

		TotalCount: 0,

		Page:       0,		return &concert.SearchResult{	BeforeEach(func() {

		PageSize:   0,

		HasMore:    false,	return nil, nil

	}, nil

}}		Events:      m.events,		cfg = &config.Config{



func (m *MockEventProvider) GetEventByID(ctx context.Context, eventID string) (*concert.Event, error) {

	return nil, concert.ErrEventNotFound

}func (m *MockEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {		TotalCount:  len(m.events),			Ticketmaster: config.TicketmasterConfig{



func (m *MockEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {	if err, exists := m.errors["getByArtist"]; exists {

	return &concert.SearchResult{

		Events:     []concert.Event{},		return nil, err		CurrentPage: 0,				ConsumerKey:    "test-key",

		TotalCount: 0,

		Page:       0,	}

		PageSize:   0,

		HasMore:    false,			TotalPages:  1,				ConsumerSecret: "test-secret",

	}, nil

}	var filteredEvents []concert.Event



func (m *MockEventProvider) GetProviderName() string {	for _, event := range m.events {		HasMore:     false,				BaseURL:        "https://app.ticketmaster.com/discovery/v2",

	return "Mock"

}		for _, artist := range event.Artists {



func (m *MockEventProvider) IsHealthy(ctx context.Context) error {			if artist.Name == artistName {	}, nil				Timeout:        30,

	return nil

}				filteredEvents = append(filteredEvents, event)

				break}				MaxResults:     200,

			}

		}				DefaultCity:    "Chicago",

	}

	func (m *MockEventProvider) GetEventByID(ctx context.Context, id string) (*concert.Event, error) {				DefaultState:   "IL",

	return &concert.SearchResult{

		Events:      filteredEvents,	if err, exists := m.errors["getById"]; exists {				DefaultCountry: "US",

		TotalCount:  len(filteredEvents),

		CurrentPage: 0,		return nil, err			},

		TotalPages:  1,

		HasMore:     false,	}		}

	}, nil

}	



func (m *MockEventProvider) IsHealthy(ctx context.Context) error {	for _, event := range m.events {		service = business.NewConcertService(cfg)

	if err, exists := m.errors["health"]; exists {

		return err		if event.ID == id {	})

	}

	return nil			return &event, nil

}

		}	Describe("Initialization", func() {

func (m *MockEventProvider) GetProviderName() string {

	return "MockProvider"	}		It("should create service with default provider", func() {

}

				Expect(service).ToNot(BeNil())

func (m *MockEventProvider) SetEvents(events []concert.Event) {

	m.events = events	return nil, nil			Expect(service.GetProviderName()).To(Equal("Ticketmaster"))

}

}		})

func (m *MockEventProvider) SetError(operation string, err error) {

	if m.errors == nil {

		m.errors = make(map[string]error)

	}func (m *MockEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {		It("should allow custom provider injection", func() {

	m.errors[operation] = err

}	if err, exists := m.errors["getByArtist"]; exists {			mockProvider := NewMockEventProvider()



func (m *MockEventProvider) AddEvent(event concert.Event) {		return nil, err			service := business.NewConcertServiceWithProvider(mockProvider, cfg)

	m.events = append(m.events, event)

}	}			

				Expect(service).ToNot(BeNil())

	var filteredEvents []concert.Event			Expect(service.GetProviderName()).To(Equal("MockProvider"))

	for _, event := range m.events {		})

		for _, artist := range event.Artists {	})

			if artist.Name == artistName {

				filteredEvents = append(filteredEvents, event)	Describe("GetUpcomingEvents", func() {

				break		It("should use default location from config", func() {

			}			ctx := context.Background()

		}			criteria := concert.SearchCriteria{}

	}			

				// This will fail with real API, but we're testing the configuration logic

	return &concert.SearchResult{			_, err := service.GetUpcomingEvents(ctx, criteria)

		Events:      filteredEvents,			

		TotalCount:  len(filteredEvents),			Expect(err).To(HaveOccurred()) // Expected since we don't have real API credentials

		CurrentPage: 0,		})

		TotalPages:  1,

		HasMore:     false,		It("should override defaults with provided criteria", func() {

	}, nil			ctx := context.Background()

}			criteria := concert.SearchCriteria{

				City:    "New York",

func (m *MockEventProvider) IsHealthy(ctx context.Context) error {				State:   "NY",

	if err, exists := m.errors["health"]; exists {				Country: "US",

		return err			}

	}			

	return nil			_, err := service.GetUpcomingEvents(ctx, criteria)

}			

			Expect(err).To(HaveOccurred()) // Expected since we don't have real API credentials

func (m *MockEventProvider) GetProviderName() string {		})

	return "MockProvider"	})

}

	Describe("SearchEventsByArtist", func() {

func (m *MockEventProvider) SetEvents(events []concert.Event) {		It("should reject empty artist name", func() {

	m.events = events			ctx := context.Background()

}			

			_, err := service.SearchEventsByArtist(ctx, "")

func (m *MockEventProvider) SetError(operation string, err error) {			

	if m.errors == nil {			Expect(err).To(HaveOccurred())

		m.errors = make(map[string]error)			Expect(err.Error()).To(ContainSubstring("artist name cannot be empty"))

	}		})

	m.errors[operation] = err

}		It("should search with valid artist name", func() {

			ctx := context.Background()

func (m *MockEventProvider) AddEvent(event concert.Event) {			

	m.events = append(m.events, event)			_, err := service.SearchEventsByArtist(ctx, "Taylor Swift")

}			
			Expect(err).To(HaveOccurred()) // Expected since we don't have real API credentials
		})
	})

	Describe("GetEventDetails", func() {
		It("should fetch event by ID", func() {
			ctx := context.Background()
			
			_, err := service.GetEventDetails(ctx, "test-event-id")
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not yet implemented"))
		})
	})

	Describe("ValidateConfiguration", func() {
		It("should validate with proper configuration", func() {
			ctx := context.Background()
			
			err := service.ValidateConfiguration(ctx)
			
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail with missing consumer key", func() {
			invalidCfg := &config.Config{
				Ticketmaster: config.TicketmasterConfig{
					ConsumerSecret: "test-secret",
					BaseURL:        "https://app.ticketmaster.com/discovery/v2",
				},
			}
			
			invalidService := business.NewConcertService(invalidCfg)
			ctx := context.Background()
			
			err := invalidService.ValidateConfiguration(ctx)
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("consumer key not configured"))
		})
	})

	Describe("GetEventsForUser", func() {
		It("should return empty result for now", func() {
			ctx := context.Background()
			
			result, err := service.GetEventsForUser(ctx, "test-user")
			
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Events).To(HaveLen(0))
			Expect(result.TotalCount).To(Equal(0))
		})
	})
})

// MockEventProvider for testing
type MockEventProvider struct {
	events  []concert.Event
	errors  map[string]error
}

func NewMockEventProvider() *MockEventProvider {
	return &MockEventProvider{
		events: []concert.Event{},
		errors: make(map[string]error),
	}
}

func (m *MockEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	if err, exists := m.errors["search"]; exists {
		return nil, err
	}
	
	return &concert.SearchResult{
		Events:      m.events,
		TotalCount:  len(m.events),
		CurrentPage: 0,
		TotalPages:  1,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) GetEventByID(ctx context.Context, id string) (*concert.Event, error) {
	if err, exists := m.errors["getById"]; exists {
		return nil, err
	}
	
	for _, event := range m.events {
		if event.ID == id {
			return &event, nil
		}
	}
	
	return nil, fmt.Errorf("event not found")
}

func (m *MockEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	if err, exists := m.errors["getByArtist"]; exists {
		return nil, err
	}
	
	var filteredEvents []concert.Event
	for _, event := range m.events {
		for _, artist := range event.Artists {
			if artist.Name == artistName {
				filteredEvents = append(filteredEvents, event)
				break
			}
		}
	}
	
	return &concert.SearchResult{
		Events:      filteredEvents,
		TotalCount:  len(filteredEvents),
		CurrentPage: 0,
		TotalPages:  1,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) IsHealthy(ctx context.Context) error {
	if err, exists := m.errors["health"]; exists {
		return err
	}
	return nil
}

func (m *MockEventProvider) GetProviderName() string {
	return "MockProvider"
}

func (m *MockEventProvider) SetEvents(events []concert.Event) {
	m.events = events
}

func (m *MockEventProvider) SetError(operation string, err error) {
	m.errors[operation] = err
}
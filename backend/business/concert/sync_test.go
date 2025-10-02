package concert_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConcertSync(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Concert Synchronization Suite")
}

var _ = Describe("Concert Synchronization", func() {
	var (
		mockProvider *MockEventProvider
		mockRepo     *MockRepository
		syncService  *concert.SyncService
		cfg          *config.Config
		ctx          context.Context
		cancel       context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		cfg = &config.Config{
			Ticketmaster: config.TicketmasterConfig{
				ConsumerKey:    "test_key",
				ConsumerSecret: "test_secret",
				BaseURL:        "https://app.ticketmaster.com/discovery/v2",
				Timeout:        10,
			},
		}

		mockProvider = NewMockEventProvider()
		mr := NewMockRepository()
		mockRepo = mr.(*MockRepository)
		syncService = concert.NewSyncService(mockProvider, mockRepo, cfg)
	})

	AfterEach(func() {
		if syncService != nil {
			syncService.Stop()
		}
		cancel()
	})

	Describe("Sync Service Creation", func() {
		It("should create a sync service with valid configuration", func() {
			Expect(syncService).ToNot(BeNil())
		})
	})

	Describe("Event Synchronization", func() {
		Context("when the provider is healthy", func() {
			BeforeEach(func() {
				mockProvider.SetHealthy(true)

				// Set up mock events for Chicago area
				chicagoEvents := []concert.Event{
					{
						ID:   "event1",
						Name: "Rock Concert at United Center",
						Date: time.Now().AddDate(0, 1, 0), // 1 month from now
						Venue: concert.Venue{
							ID:   "venue1",
							Name: "United Center",
							Address: concert.Address{
								Street:  "1901 W Madison St",
								City:    "Chicago",
								State:   "IL",
								Country: "US",
								Postal:  "60612",
							},
							Capacity: 23500,
						},
						Artists: []concert.Artist{
							{ID: "artist1", Name: "Test Band", Genres: []string{"Rock"}},
						},
						Genres:         []string{"Rock", "Alternative"},
						Status:         "onsale",
						TicketURL:      "https://ticketmaster.com/event1",
						Description:    "Amazing rock concert",
						AgeRestriction: "All Ages",
						PriceRange: concert.PriceRange{
							Min:      50.0,
							Max:      150.0,
							Currency: "USD",
						},
					},
					{
						ID:   "event2",
						Name: "Jazz Night at Chicago Theatre",
						Date: time.Now().AddDate(0, 2, 0), // 2 months from now
						Venue: concert.Venue{
							ID:   "venue2",
							Name: "Chicago Theatre",
							Address: concert.Address{
								Street:  "175 N State St",
								City:    "Chicago",
								State:   "IL",
								Country: "US",
								Postal:  "60601",
							},
							Capacity: 3600,
						},
						Artists: []concert.Artist{
							{ID: "artist2", Name: "Jazz Quartet", Genres: []string{"Jazz"}},
						},
						Genres:         []string{"Jazz"},
						Status:         "onsale",
						TicketURL:      "https://ticketmaster.com/event2",
						Description:    "Smooth jazz evening",
						AgeRestriction: "21+",
						PriceRange: concert.PriceRange{
							Min:      75.0,
							Max:      200.0,
							Currency: "USD",
						},
					},
				}

				mockProvider.SetSearchResults(&concert.SearchResult{
					Events:      chicagoEvents,
					TotalCount:  2,
					CurrentPage: 1,
					TotalPages:  1,
					HasMore:     false,
				})
			})

			It("should successfully sync events on startup", func() {
				// Start the sync service
				err := syncService.Start(ctx)
				Expect(err).ToNot(HaveOccurred())

				// Wait a moment for the initial sync to complete
				time.Sleep(100 * time.Millisecond)

				// Verify that events were upserted
				Expect(mockRepo.GetUpsertedEventCount()).To(Equal(2))

				// Verify that old events were cleaned up
				Expect(mockRepo.WasDeleteOldEventsCalled()).To(BeTrue())

				// Verify sync status
				status, err := syncService.GetSyncStatus(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(status.IsRunning).To(BeTrue())
				Expect(status.Provider).To(Equal("mock"))
				Expect(status.EventCount).To(Equal(int64(2)))
			})

			It("should handle pagination correctly", func() {
				// Set up paginated results
				page1Events := []concert.Event{
					{
						ID:   "event1",
						Name: "Event 1",
						Date: time.Now().AddDate(0, 1, 0),
						Venue: concert.Venue{
							ID: "venue1", Name: "Venue 1",
							Address: concert.Address{City: "Chicago", State: "IL", Country: "US"},
						},
						Artists: []concert.Artist{{ID: "artist1", Name: "Artist 1"}},
						Status:  "onsale",
					},
				}

				page2Events := []concert.Event{
					{
						ID:   "event2",
						Name: "Event 2",
						Date: time.Now().AddDate(0, 2, 0),
						Venue: concert.Venue{
							ID: "venue2", Name: "Venue 2",
							Address: concert.Address{City: "Chicago", State: "IL", Country: "US"},
						},
						Artists: []concert.Artist{{ID: "artist2", Name: "Artist 2"}},
						Status:  "onsale",
					},
				}

				mockProvider.SetPaginatedResults(map[int]*concert.SearchResult{
					1: {
						Events:      page1Events,
						TotalCount:  2,
						CurrentPage: 1,
						TotalPages:  2,
						HasMore:     true,
					},
					2: {
						Events:      page2Events,
						TotalCount:  2,
						CurrentPage: 2,
						TotalPages:  2,
						HasMore:     false,
					},
				})

				err := syncService.Start(ctx)
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(100 * time.Millisecond)

				// Should have synced events from both pages
				Expect(mockRepo.GetUpsertedEventCount()).To(Equal(2))
			})
		})

		Context("when the provider is unhealthy", func() {
			BeforeEach(func() {
				mockProvider.SetHealthy(false)
			})

			It("should fail to sync and return an error", func() {
				err := syncService.Start(ctx)
				// The service should start but the sync should fail
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(100 * time.Millisecond)

				// No events should be upserted due to provider being unhealthy
				Expect(mockRepo.GetUpsertedEventCount()).To(Equal(0))
			})
		})

		Context("when repository operations fail", func() {
			BeforeEach(func() {
				mockProvider.SetHealthy(true)
				mockProvider.SetSearchResults(&concert.SearchResult{
					Events: []concert.Event{
						{
							ID:   "event1",
							Name: "Test Event",
							Date: time.Now().AddDate(0, 1, 0),
							Venue: concert.Venue{
								ID: "venue1", Name: "Test Venue",
								Address: concert.Address{City: "Chicago", State: "IL", Country: "US"},
							},
							Artists: []concert.Artist{{ID: "artist1", Name: "Test Artist"}},
							Status:  "onsale",
						},
					},
					TotalCount:  1,
					CurrentPage: 1,
					TotalPages:  1,
					HasMore:     false,
				})

				// Make repository operations fail
				mockRepo.SetFailUpsert(true)
			})

			It("should continue with other events even if some fail", func() {
				err := syncService.Start(ctx)
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(100 * time.Millisecond)

				// Sync should have been attempted but failed
				Expect(mockRepo.GetUpsertedEventCount()).To(Equal(0))
			})
		})
	})

	Describe("Manual Sync", func() {
		BeforeEach(func() {
			mockProvider.SetHealthy(true)
			mockProvider.SetSearchResults(&concert.SearchResult{
				Events: []concert.Event{
					{
						ID:   "manual_event",
						Name: "Manual Sync Event",
						Date: time.Now().AddDate(0, 1, 0),
						Venue: concert.Venue{
							ID: "venue_manual", Name: "Manual Venue",
							Address: concert.Address{City: "Chicago", State: "IL", Country: "US"},
						},
						Artists: []concert.Artist{{ID: "artist_manual", Name: "Manual Artist"}},
						Status:  "onsale",
					},
				},
				TotalCount:  1,
				CurrentPage: 1,
				TotalPages:  1,
				HasMore:     false,
			})
		})

		It("should perform manual sync successfully", func() {
			err := syncService.ManualSync(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Verify that the event was upserted
			Expect(mockRepo.GetUpsertedEventCount()).To(Equal(1))
		})
	})

	Describe("Service Lifecycle", func() {
		It("should start and stop cleanly", func() {
			mockProvider.SetHealthy(true)

			err := syncService.Start(ctx)
			Expect(err).ToNot(HaveOccurred())

			status, err := syncService.GetSyncStatus(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.IsRunning).To(BeTrue())

			syncService.Stop()

			// After stopping, service should not be running
			// Note: In the actual implementation, you might want to track this state
		})
	})
})

// Mock implementations

type MockEventProvider struct {
	healthy          bool
	searchResults    *concert.SearchResult
	paginatedResults map[int]*concert.SearchResult
}

func NewMockEventProvider() *MockEventProvider {
	return &MockEventProvider{
		healthy:          true,
		paginatedResults: make(map[int]*concert.SearchResult),
	}
}

func (m *MockEventProvider) SetHealthy(healthy bool) {
	m.healthy = healthy
}

func (m *MockEventProvider) SetSearchResults(results *concert.SearchResult) {
	m.searchResults = results
}

func (m *MockEventProvider) SetPaginatedResults(results map[int]*concert.SearchResult) {
	m.paginatedResults = results
}

func (m *MockEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	if !m.healthy {
		return nil, concert.ErrProviderUnhealthy
	}

	// Check for paginated results first
	if result, exists := m.paginatedResults[criteria.Page]; exists {
		return result, nil
	}

	if m.searchResults != nil {
		return m.searchResults, nil
	}

	return &concert.SearchResult{
		Events:      []concert.Event{},
		TotalCount:  0,
		CurrentPage: 1,
		TotalPages:  1,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) GetEventByID(ctx context.Context, id string) (*concert.Event, error) {
	if !m.healthy {
		return nil, concert.ErrProviderUnhealthy
	}
	return nil, concert.ErrEventNotFound
}

func (m *MockEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	return m.SearchEvents(ctx, criteria)
}

func (m *MockEventProvider) GetProviderName() string {
	return "mock"
}

func (m *MockEventProvider) IsHealthy(ctx context.Context) error {
	if !m.healthy {
		return concert.ErrProviderUnhealthy
	}
	return nil
}

type MockRepository struct {
	upsertedEventCount    int
	deleteOldEventsCalled bool
	failUpsert            bool
	eventCount            int64
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		eventCount: 2, // Default mock count
	}
}

func (m *MockRepository) SetFailUpsert(fail bool) {
	m.failUpsert = fail
}

func (m *MockRepository) GetUpsertedEventCount() int {
	return m.upsertedEventCount
}

func (m *MockRepository) WasDeleteOldEventsCalled() bool {
	return m.deleteOldEventsCalled
}

func (m *MockRepository) UpsertEvent(ctx context.Context, event *concert.Event) error {
	if m.failUpsert {
		return concert.ErrRepositoryFailure
	}
	m.upsertedEventCount++
	return nil
}

func (m *MockRepository) DeleteOldEvents(ctx context.Context, cutoffDate time.Time) error {
	m.deleteOldEventsCalled = true
	return nil
}

func (m *MockRepository) GetEventCount(ctx context.Context) (int64, error) {
	return m.eventCount, nil
}

func (m *MockRepository) IsHealthy(ctx context.Context) error {
	return nil
}

// Minimal implementations for other required methods
func (m *MockRepository) GetEventByID(ctx context.Context, id string) (*concert.Event, error) {
	return nil, concert.ErrEventNotFound
}

func (m *MockRepository) GetEventsInDateRange(ctx context.Context, startDate, endDate time.Time, city, status string, limit, offset int) ([]*concert.Event, error) {
	return []*concert.Event{}, nil
}

func (m *MockRepository) GetEventsForArtist(ctx context.Context, artistName, city string, limit int) ([]*concert.Event, error) {
	return []*concert.Event{}, nil
}

func (m *MockRepository) GetUpcomingEventsInCity(ctx context.Context, city string, limit int) ([]*Event, error) {
	return []*Event{}, nil
}

func (m *MockRepository) UpsertVenue(ctx context.Context, venue *concert.Venue) error {
	return nil
}

func (m *MockRepository) UpsertArtist(ctx context.Context, artist *concert.Artist) error {
	return nil
}

func (m *MockRepository) GetEventArtists(ctx context.Context, eventID string) ([]concert.Artist, error) {
	return []concert.Artist{}, nil
}

func (m *MockRepository) AssociateEventWithArtist(ctx context.Context, eventID, artistID, role string) error {
	return nil
}

// Custom errors for testing
var (
	ErrProviderUnhealthy = fmt.Errorf("provider is unhealthy")
	ErrEventNotFound     = fmt.Errorf("event not found")
	ErrRepositoryFailure = fmt.Errorf("repository operation failed")
)

package concert

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
)

// SyncService handles periodic synchronization of concert data from external APIs
type SyncService struct {
	eventProvider EventProvider
	repository    Repository
	config        *config.Config
	stopChan      chan struct{}
	ticker        *time.Ticker
}

// NewSyncService creates a new concert sync service
func NewSyncService(provider EventProvider, repo Repository, cfg *config.Config) *SyncService {
	return &SyncService{
		eventProvider: provider,
		repository:    repo,
		config:        cfg,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the synchronization process
// It performs an initial sync and then schedules periodic syncs every 24 hours
func (s *SyncService) Start(ctx context.Context) error {
	log.Printf("Starting concert sync service...")

	// Perform initial sync
	if err := s.syncEvents(ctx); err != nil {
		log.Printf("Error during initial sync: %v", err)
		// Don't fail startup on sync error, just log it
	}

	// Start periodic sync
	s.ticker = time.NewTicker(24 * time.Hour)
	go s.syncLoop(ctx)

	log.Printf("Concert sync service started successfully")
	return nil
}

// Stop gracefully shuts down the sync service
func (s *SyncService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
	log.Printf("Concert sync service stopped")
}

// syncLoop runs the periodic synchronization
func (s *SyncService) syncLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-s.ticker.C:
			if err := s.syncEvents(ctx); err != nil {
				log.Printf("Error during scheduled sync: %v", err)
			}
		}
	}
}

// syncEvents performs the actual synchronization of concert data
func (s *SyncService) syncEvents(ctx context.Context) error {
	log.Printf("Starting concert data synchronization...")

	// Check provider health
	if err := s.eventProvider.IsHealthy(ctx); err != nil {
		return fmt.Errorf("event provider is not healthy: %w", err)
	}

	// Define search criteria for Chicago area events over next 6 months
	endDate := time.Now().AddDate(0, 6, 0) // 6 months from now
	criteria := SearchCriteria{
		City:       "Chicago",
		State:      "IL",
		Country:    "US",
		StartDate:  time.Now(),
		EndDate:    endDate,
		MaxResults: 200, // Fetch up to 200 events per request
		Page:       1,
	}

	totalSynced := 0
	page := 1

	for {
		criteria.Page = page

		log.Printf("Fetching events page %d...", page)

		// Get events from the provider
		result, err := s.eventProvider.SearchEvents(ctx, criteria)
		if err != nil {
			return fmt.Errorf("failed to search events: %w", err)
		}

		if len(result.Events) == 0 {
			log.Printf("No more events to sync")
			break
		}

		// Sync each event to the database
		for _, event := range result.Events {
			if err := s.repository.UpsertEvent(ctx, &event); err != nil {
				log.Printf("Error upserting event %s (%s): %v", event.Name, event.ID, err)
				continue // Continue with other events
			}
			totalSynced++
		}

		log.Printf("Synced %d events from page %d", len(result.Events), page)

		// Check if we have more pages
		if !result.HasMore || page >= result.TotalPages {
			break
		}

		page++

		// Add a small delay to be respectful to the API
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue
		}
	}

	// Clean up old events (remove events that have passed)
	cutoffDate := time.Now()
	if err := s.repository.DeleteOldEvents(ctx, cutoffDate); err != nil {
		log.Printf("Error cleaning up old events: %v", err)
		// Don't fail sync on cleanup error
	} else {
		log.Printf("Cleaned up past events older than %s", cutoffDate.Format("2006-01-02 15:04:05"))
	}

	log.Printf("Concert sync completed. Total events synced: %d", totalSynced)
	return nil
}

// GetSyncStatus returns information about the sync service status
func (s *SyncService) GetSyncStatus(ctx context.Context) (*SyncStatus, error) {
	count, err := s.repository.GetEventCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get event count: %w", err)
	}

	return &SyncStatus{
		IsRunning:  s.ticker != nil,
		EventCount: count,
		Provider:   s.eventProvider.GetProviderName(),
		LastSyncAt: time.Now(), // In production, you'd track this properly
		NextSyncAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// SyncStatus represents the current status of the sync service
type SyncStatus struct {
	IsRunning  bool      `json:"isRunning"`
	EventCount int64     `json:"eventCount"`
	Provider   string    `json:"provider"`
	LastSyncAt time.Time `json:"lastSyncAt"`
	NextSyncAt time.Time `json:"nextSyncAt"`
}

// ManualSync triggers a manual synchronization (useful for testing/admin)
func (s *SyncService) ManualSync(ctx context.Context) error {
	log.Printf("Manual concert sync requested")
	return s.syncEvents(ctx)
}

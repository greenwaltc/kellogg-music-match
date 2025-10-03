package concert

import (
	"context"
	"fmt"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
	"github.com/greenwaltc/kellogg-music-match/backend/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// SyncService handles periodic synchronization of concert data from external APIs
type SyncService struct {
	eventProvider EventProvider
	repository    Repository
	config        *config.Config
	stopChan      chan struct{}
	ticker        *time.Ticker
	stopped       bool
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
	logger.FromCtx(ctx).Info("sync starting")

	// Perform initial sync
	if err := s.syncEvents(ctx); err != nil {
		logger.FromCtx(ctx).Error("sync initial failed", "error", err)
		// Don't fail startup on sync error, just log it
	}

	// Start periodic sync
	s.ticker = time.NewTicker(24 * time.Hour)
	go s.syncLoop(ctx)

	logger.FromCtx(ctx).Info("sync started")
	return nil
}

// Stop gracefully shuts down the sync service
func (s *SyncService) Stop() {
	if s.stopped {
		return
	}
	s.stopped = true
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
	logger.FromCtx(context.Background()).Info("sync stopped")
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
				logger.FromCtx(ctx).Error("sync scheduled failed", "error", err)
			}
		}
	}
}

// syncEvents performs the actual synchronization of concert data
func (s *SyncService) syncEvents(ctx context.Context) error {
	logger.FromCtx(ctx).Info("sync cycle begin")
	tracer := otel.Tracer("concert.sync")
	ctx, span := tracer.Start(ctx, "syncEvents")
	defer span.End()

	// Check provider health
	if err := s.eventProvider.IsHealthy(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		telemetry.SyncCycleCounter.WithLabelValues("provider_unhealthy").Inc()
		return fmt.Errorf("event provider is not healthy: %w", err)
	}

	// Define search criteria for Chicago area events over configured date range
	endDate := time.Now().AddDate(0, s.config.Ticketmaster.DateRangeMonths, 0)
	criteria := SearchCriteria{
		City:       s.config.Ticketmaster.DefaultCity,
		State:      s.config.Ticketmaster.DefaultState,
		Country:    s.config.Ticketmaster.DefaultCountry,
		StartDate:  time.Now(),
		EndDate:    endDate,
		MaxResults: s.config.Ticketmaster.MaxResults, // Use configured max results
		Page:       1,
	}

	totalSynced := 0
	page := 1

	for {
		criteria.Page = page

		logger.FromCtx(ctx).Debug("sync fetching page", "page", page)

		// Get events from the provider
		result, err := s.eventProvider.SearchEvents(ctx, criteria)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			telemetry.SyncCycleCounter.WithLabelValues("search_error").Inc()
			return fmt.Errorf("failed to search events: %w", err)
		}

		if len(result.Events) == 0 {
			logger.FromCtx(ctx).Info("sync no more events")
			break
		}

		// Sync each event to the database
		for _, event := range result.Events {
			if err := s.repository.UpsertEvent(ctx, &event); err != nil {
				logger.FromCtx(ctx).Warn("event upsert failed", "eventId", event.ID, "name", event.Name, "error", err)
				continue // Continue with other events
			}
			totalSynced++
		}

		logger.FromCtx(ctx).Info("page synced", "count", len(result.Events), "page", page)
		span.SetAttributes(attribute.Int("page", page), attribute.Int("events_on_page", len(result.Events)))

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
		logger.FromCtx(ctx).Warn("cleanup old events failed", "error", err)
		// Don't fail sync on cleanup error
	} else {
		logger.FromCtx(ctx).Debug("cleanup old events success", "cutoff", cutoffDate.Format(time.RFC3339))
	}

	logger.FromCtx(ctx).Info("sync cycle complete", "totalSynced", totalSynced)
	telemetry.SyncEventsCounter.Add(float64(totalSynced))
	telemetry.SyncCycleCounter.WithLabelValues("success").Inc()
	span.SetAttributes(attribute.Int("totalSynced", totalSynced))
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
	logger.FromCtx(ctx).Info("manual sync requested")
	return s.syncEvents(ctx)
}

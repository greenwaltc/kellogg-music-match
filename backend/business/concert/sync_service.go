package concert

import (
	"context"
	"fmt"
	"strings"
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

	// Define the full search window; we'll segment it to avoid deep paging caps
	fullStartUTC := time.Now().UTC()
	months := s.config.Ticketmaster.DateRangeMonths
	if months <= 0 {
		months = 12 // default to a year if misconfigured; proxy will clamp as needed
	}
	fullEndUTC := fullStartUTC.AddDate(0, months, 0)
	span.SetAttributes(
		attribute.String("sync.startUTC", fullStartUTC.Format(time.RFC3339)),
		attribute.String("sync.endUTC", fullEndUTC.UTC().Format(time.RFC3339)),
	)

	criteria := SearchCriteria{
		City:       s.config.Ticketmaster.DefaultCity,
		State:      s.config.Ticketmaster.DefaultState,
		Country:    s.config.Ticketmaster.DefaultCountry,
		StartDate:  fullStartUTC,
		EndDate:    fullEndUTC,
		MaxResults: s.config.Ticketmaster.MaxResults, // Use configured max results
		Page:       1,
	}

	totalSynced := 0

	// Segment the full window into 6-month chunks to avoid deep paging (page*size < 1000)
	segmentMonths := 6
	if months < segmentMonths {
		segmentMonths = months
	}
	segmentStart := fullStartUTC
	segmentIndex := 0
	for segmentStart.Before(fullEndUTC) {
		segmentEnd := segmentStart.AddDate(0, segmentMonths, 0)
		if segmentEnd.After(fullEndUTC) {
			segmentEnd = fullEndUTC
		}

		// Pin the context window for this segment
		segCtx := context.WithValue(ctx, tmStartKey, segmentStart)
		segCtx = context.WithValue(segCtx, tmEndKey, segmentEnd)
		logger.FromCtx(segCtx).Info("sync segment begin", "segment", segmentIndex, "start", segmentStart.Format(time.RFC3339), "end", segmentEnd.Format(time.RFC3339))
		span.SetAttributes(
			attribute.Int("sync.segment.index", segmentIndex),
			attribute.String("sync.segment.startUTC", segmentStart.Format(time.RFC3339)),
			attribute.String("sync.segment.endUTC", segmentEnd.Format(time.RFC3339)),
		)

		// Apply segment-specific date window to search criteria
		criteria.StartDate = segmentStart
		criteria.EndDate = segmentEnd

		// Reset pagination for this segment
		page := 1
		for {
			criteria.Page = page
			logger.FromCtx(segCtx).Debug("sync fetching page", "page", page)

			// Ticketmaster deep paging cap: (pageIndex * size) must be < 1000 (pageIndex is 0-based)
			// If the next request would exceed the cap, stop gracefully to avoid 400 DIS1035
			size := s.config.Ticketmaster.MaxResults
			if size <= 0 {
				size = 100
			}
			if (page-1)*size >= 1000 {
				// record on span and log
				span.SetAttributes(attribute.Int("ticketmaster.depth_cap_size", size), attribute.Int("ticketmaster.depth_cap_page", page))
				logger.FromCtx(ctx).Warn("stopping before Ticketmaster deep paging cap", "page", page, "size", size)
				break
			}

			// Get events from the provider
			result, err := s.eventProvider.SearchEvents(segCtx, criteria)
			if err != nil {
				// If a later page returns 400, attempt a single retry with additional delay; if it still 400s, treat as end-of-pages
				if page > 1 && strings.Contains(err.Error(), "status 400") {
					// backoff: 2x configured delay + small jitter
					extraDelay := time.Duration(s.config.Ticketmaster.PageDelayMs*2)*time.Millisecond + time.Duration((page%5)*20)*time.Millisecond
					logger.FromCtx(segCtx).Warn("page fetch returned 400, retrying once", "page", page, "delayMs", int(extraDelay/time.Millisecond))
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(extraDelay):
					}
					result, err = s.eventProvider.SearchEvents(segCtx, criteria)
					if err != nil {
						if strings.Contains(err.Error(), "status 400") {
							logger.FromCtx(segCtx).Warn("pagination terminated due to API 400 after retry", "page", page)
							break
						}
						span.RecordError(err)
						span.SetStatus(codes.Error, err.Error())
						telemetry.SyncCycleCounter.WithLabelValues("search_error").Inc()
						return fmt.Errorf("failed to search events: %w", err)
					}
				} else {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					telemetry.SyncCycleCounter.WithLabelValues("search_error").Inc()
					return fmt.Errorf("failed to search events: %w", err)
				}
			}

			if len(result.Events) == 0 {
				logger.FromCtx(segCtx).Info("sync no more events (segment)")
				break
			}

			// Sync each event to the database
			for _, event := range result.Events {
				if err := s.repository.UpsertEvent(ctx, &event); err != nil {
					logger.FromCtx(segCtx).Warn("event upsert failed", "eventId", event.ID, "name", event.Name, "error", err)
					continue // Continue with other events
				}
				totalSynced++
			}

			logger.FromCtx(segCtx).Info("page synced", "count", len(result.Events), "page", page)
			span.SetAttributes(attribute.Int("page", page), attribute.Int("events_on_page", len(result.Events)))

			// Check if we have more pages
			if !result.HasMore || page >= result.TotalPages {
				break
			}

			page++

			// Add a small delay to respect Ticketmaster's rate limit (~5 req/s)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(s.config.Ticketmaster.PageDelayMs)*time.Millisecond + time.Duration((page%5)*10)*time.Millisecond):
				// Continue
			}
		}

		// Advance to next segment
		segmentStart = segmentEnd
		segmentIndex++
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

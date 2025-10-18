package business

import (
	"context"
	"time"

	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
	"github.com/jackc/pgx/v5/pgtype"
)

// CleanupQuerier defines the minimal sqlc methods used by the cleanup service
type CleanupQuerier interface {
	DeletePastEvents(ctx context.Context, cutoffDate pgtype.Timestamptz) error
	DeleteAllOrphanedEvents(ctx context.Context) error
	DeleteOldConcertEvents(ctx context.Context, cutoffDate pgtype.Timestamp) error
}

// CleanupService periodically removes past events and orphaned events
type CleanupService struct {
	q       CleanupQuerier
	ticker  *time.Ticker
	stopCh  chan struct{}
	started bool
}

func NewCleanupService(q CleanupQuerier) *CleanupService {
	return &CleanupService{q: q, stopCh: make(chan struct{})}
}

// Start runs cleanup once at startup and every 24 hours thereafter
func (s *CleanupService) Start(ctx context.Context) {
	if s.started {
		return
	}
	s.started = true
	logger.FromCtx(ctx).Info("cleanup service starting")
	s.runOnce(ctx)
	s.ticker = time.NewTicker(24 * time.Hour)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-s.ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

// Stop stops the periodic cleanup
func (s *CleanupService) Stop() {
	if !s.started {
		return
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopCh)
}

// CleanOnce exposes single-run cleanup for tests
func (s *CleanupService) CleanOnce(ctx context.Context) {
	s.runOnce(ctx)
}

func (s *CleanupService) runOnce(ctx context.Context) {
	now := time.Now().UTC()
	// Past on-demand events: start_utc older than now
	if err := s.q.DeletePastEvents(ctx, pgtype.Timestamptz{Time: now, Valid: true}); err != nil {
		logger.FromCtx(ctx).Warn("cleanup DeletePastEvents failed", "error", err)
	} else {
		logger.FromCtx(ctx).Debug("cleanup DeletePastEvents success", "cutoff", now)
	}
	// Orphaned events: no associations
	if err := s.q.DeleteAllOrphanedEvents(ctx); err != nil {
		logger.FromCtx(ctx).Warn("cleanup DeleteAllOrphanedEvents failed", "error", err)
	} else {
		logger.FromCtx(ctx).Debug("cleanup DeleteAllOrphanedEvents success")
	}
	// Legacy concert_events cleanup remains for back-compat until decommissioned
	if err := s.q.DeleteOldConcertEvents(ctx, pgtype.Timestamp{Time: now, Valid: true}); err != nil {
		logger.FromCtx(ctx).Warn("cleanup DeleteOldConcertEvents failed", "error", err)
	} else {
		logger.FromCtx(ctx).Debug("cleanup DeleteOldConcertEvents success", "cutoff", now)
	}
}

// Ensure sqlc.Queries satisfies CleanupQuerier
var _ CleanupQuerier = (*sqlc.Queries)(nil)

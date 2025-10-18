package business

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type fakeCleanupQuerier struct {
	pastCalled       bool
	orphanCalled     bool
	legacyCalled     bool
	lastPastCutoff   pgtype.Timestamptz
	lastLegacyCutoff pgtype.Timestamp
}

func (f *fakeCleanupQuerier) DeletePastEvents(ctx context.Context, cutoffDate pgtype.Timestamptz) error {
	f.pastCalled = true
	f.lastPastCutoff = cutoffDate
	return nil
}
func (f *fakeCleanupQuerier) DeleteAllOrphanedEvents(ctx context.Context) error {
	f.orphanCalled = true
	return nil
}
func (f *fakeCleanupQuerier) DeleteOldConcertEvents(ctx context.Context, cutoffDate pgtype.Timestamp) error {
	f.legacyCalled = true
	f.lastLegacyCutoff = cutoffDate
	return nil
}

func TestCleanupService_CleanOnce(t *testing.T) {
	fq := &fakeCleanupQuerier{}
	svc := NewCleanupService(fq)
	// Run a single cleanup
	svc.CleanOnce(context.Background())
	require.True(t, fq.pastCalled)
	require.True(t, fq.orphanCalled)
	require.True(t, fq.legacyCalled)
	// Ensure cutoffs are set as valid timestamps
	require.True(t, fq.lastPastCutoff.Valid)
	require.True(t, fq.lastLegacyCutoff.Valid)
	// Cutoffs should be near-now
	now := time.Now().UTC()
	require.WithinDuration(t, now, fq.lastPastCutoff.Time, 2*time.Second)
	require.WithinDuration(t, now, fq.lastLegacyCutoff.Time, 2*time.Second)
}

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/spotify"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// minimal in-memory auth + matching dependency placeholders

type dummyMatching struct{}

func TestSyncSpotifyFlow(t *testing.T) {
	if os.Getenv("SPOTIFY_CLIENT_ID") == "" || os.Getenv("SPOTIFY_CLIENT_SECRET") == "" {
		t.Skip("skipping Spotify sync integration test: SPOTIFY_CLIENT_ID/SECRET not set")
	}
	// Build dependencies
	spotifyService := spotify.NewService(nil, "")            // disable persistence for integration test
	matchingService := business.NewMatchingService(nil, nil) // may be nil-safe for our purposes (if not, we’d adjust)
	wrapper := NewMatchingAPIServiceWrapper(matchingService, spotifyService).(*MatchingAPIServiceWrapper)

	// Simulate user context
	ctx := context.WithValue(context.Background(), UserContextKey, &UserContext{UserID: "u1", Username: "alice"})

	// Start sync
	resp, err := wrapper.SyncSpotify(ctx, generated.SpotifySyncStartRequest{Code: "code123", State: "stateXYZ"})
	if err != nil {
		t.Fatalf("sync start error: %v", err)
	}
	if resp.Code != 202 {
		t.Fatalf("expected 202, got %d", resp.Code)
	}

	// Query status immediately
	statusResp, err := wrapper.GetSpotifySyncStatus(ctx)
	if err != nil {
		t.Fatalf("status error: %v", err)
	}
	if statusResp.Code != 200 {
		t.Fatalf("expected 200 status code, got %d", statusResp.Code)
	}
	body := statusResp.Body.(generated.SpotifySyncStatusResponse)
	if body.Status != spotify.StatusPending && body.Status != spotify.StatusInProgress {
		t.Fatalf("unexpected initial status: %s", body.Status)
	}

	// Poll until complete (bounded)
	for i := 0; i < 30; i++ { // ~30 * 250ms = 7.5s max
		statusResp, _ = wrapper.GetSpotifySyncStatus(ctx)
		body = statusResp.Body.(generated.SpotifySyncStatusResponse)
		if body.Status == spotify.StatusComplete {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("sync did not complete within expected polling window; last status=%s progress=%d", body.Status, body.Progress)
}

// Example of hitting the generated router end-to-end would require constructing all controllers; omitted for brevity.
// We still touch wrapper logic which is where custom Spotify integration resides.

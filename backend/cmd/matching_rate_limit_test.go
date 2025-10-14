package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// TestFindMusicMatchesRateLimit validates per-user rate limiting returns 429 after threshold.
func TestFindMusicMatchesRateLimit(t *testing.T) {
	ResetMatchRateLimiter()
	// Build minimal app wiring by reusing existing main composition if available.
	// Here we directly call the generated controller through wrapper since full server setup may be heavy.
	// We assume NewMatchingAPIServiceWrapper is used in main; reuse existing matchingService and spotifyService via a lightweight test harness.

	// We call the wrapper directly instead of full HTTP stack to focus on limiter behavior.
	// Provide a dummy matchingService that is only invoked for first 3 (non-rate-limited) calls. It may return a simple OK response.
	mw := &MatchingAPIServiceWrapper{matchingService: &dummyMatchingService{}, spotifyService: nil}

	call := func(username string) (generated.ImplResponse, context.Context) {
		body := map[string]interface{}{"artists": []string{"A"}}
		b, _ := json.Marshal(body)
		// minimal decode into ArtistsRequest
		var ar generated.ArtistsRequest
		_ = json.Unmarshal(b, &ar)
		ctx := context.WithValue(context.Background(), UserContextKey, &UserContext{Username: username, UserID: username})
		resp, _ := mw.FindMusicMatches(ctx, ar, username, "medium_term", "artists", 10, 0)
		return resp, ctx
	}

	for i := 1; i <= 3; i++ {
		resp, _ := call("user1")
		if resp.Code == 429 {
			t.Fatalf("unexpected 429 on attempt %d", i)
		}
	}
	resp, _ := call("user1")
	if resp.Code != 429 {
		t.Fatalf("expected 429 on 4th attempt, got %d", resp.Code)
	}

	// After window passes (sleep >10s) reset effect naturally; we shorten by directly manipulating time not implemented here — skip to keep test fast.
	_ = time.Now()
}

type dummyMatchingService struct{}

func (d *dummyMatchingService) FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, username string, range_ string, limit int32, overlapsLimit ...int32) (generated.ImplResponse, error) {
	return generated.Response(200, []interface{}{}), nil
}

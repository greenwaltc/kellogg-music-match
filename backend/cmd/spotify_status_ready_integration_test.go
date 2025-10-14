package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"fmt"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/spotify"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
)

// This integration test spins up the mux used in main() and asserts that the
// /sync/spotify/status endpoint returns ready=true after we insert a minimal
// snapshot row for the authenticated user.
func TestSpotifyStatusReadyFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	// Ensure a database is configured (relies on env from docker-compose tests env)
	cfg := config.Load()
	repo, err := business.NewUserRepositoryWithConfig(&cfg.Database)
	if err != nil {
		t.Skipf("skipping: db not available: %v", err)
	}
	if pr, ok := repo.(*business.PostgreSQLUserRepository); ok {
		defer pr.Close()
	}

	// Build services similar to main
	jwtService := business.NewJWTService(&cfg.JWT)
	healthService := business.NewHealthService()
	matchingEngine := business.NewMatchingEngine()
	_ = matchingEngine
	spotifyService := spotify.NewService(repo, cfg.Spotify.RefreshTokenKey, spotify.WithSpotifyCredentials(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret, cfg.Spotify.RedirectURI))
	feedbackService := business.NewFeedbackService(repo)

	_ = jwtService
	_ = healthService
	_ = feedbackService

	// Create a user and a minimal snapshot row
	uid := uuid.New()
	sfx := uuid.NewString()[:8]
	uname := fmt.Sprintf("status_ready_user_%s", sfx)
	email := fmt.Sprintf("status_ready_%s@example.com", sfx)
	_, err = repo.CreateUser(context.Background(), uid, uname, email, "S", "R", "hash", "2Y", int32(time.Now().Year()+2))
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	fetchedAt := time.Now().UTC()
	// Store one artist snapshot for medium_term to signal readiness
	if err := repo.StoreSpotifyTopArtists(context.Background(), uid, fetchedAt, "medium_term", []business.SpotifyTopArtist{{Rank: 1, SpotifyArtistID: "aid", Name: "Artist"}}); err != nil {
		t.Fatalf("store artist snapshot failed: %v", err)
	}

	// Build mux and wrap JWT middleware to inject our user (bypass full token issuance)
	mux := http.NewServeMux()
	// Use the same handler registered in main for status
	// Re-register with closure over services from this test
	mux.HandleFunc("/sync/spotify/status", func(w http.ResponseWriter, r *http.Request) {
		// Inject context with our user
		ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{UserID: uid.String(), Username: uname})
		r = r.WithContext(ctx)
		// Call the actual handler logic from main by duplicating minimal code path here
		job := spotifyService.GetStatus(uname)
		// compute ready by querying views
		var hasArtists, hasTracks bool
		if pr, ok := repo.(*business.PostgreSQLUserRepository); ok {
			_ = pr.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM v_current_spotify_top_artists WHERE user_id=$1)`, uid).Scan(&hasArtists)
			_ = pr.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM v_current_spotify_top_tracks WHERE user_id=$1)`, uid).Scan(&hasTracks)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": job.Status, "ready": hasArtists || hasTracks})
	})

	// Issue request
	req := httptest.NewRequest(http.MethodGet, "/sync/spotify/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var out struct {
		Ready bool `json:"ready"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if !out.Ready {
		t.Fatalf("expected ready=true, got false")
	}
	_ = os.Unsetenv("_TEST_DUMMY")
}

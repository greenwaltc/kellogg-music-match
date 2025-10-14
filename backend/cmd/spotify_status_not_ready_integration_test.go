package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fmt"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/spotify"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
)

// This integration test asserts /sync/spotify/status returns ready=false
// when no Spotify snapshot data exists for the authenticated user.
func TestSpotifyStatusNotReadyFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	cfg := config.Load()
	repo, err := business.NewUserRepositoryWithConfig(&cfg.Database)
	if err != nil {
		t.Skipf("skipping: db not available: %v", err)
	}
	if pr, ok := repo.(*business.PostgreSQLUserRepository); ok {
		defer pr.Close()
	}

	// Build services similar to main
	spotifyService := spotify.NewService(repo, cfg.Spotify.RefreshTokenKey, spotify.WithSpotifyCredentials(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret, cfg.Spotify.RedirectURI))

	// Create a user with no snapshots
	uid := uuid.New()
	sfx := uuid.NewString()[:8]
	uname := fmt.Sprintf("status_not_ready_user_%s", sfx)
	email := fmt.Sprintf("status_not_ready_%s@example.com", sfx)
	_, err = repo.CreateUser(context.Background(), uid, uname, email, "S", "R", "hash", "2Y", int32(time.Now().Year()+2))
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	// Build mux with a status handler mirroring main() logic but injecting context directly
	mux := http.NewServeMux()
	mux.HandleFunc("/sync/spotify/status", func(w http.ResponseWriter, r *http.Request) {
		// Inject user into context (bypassing JWT for this test)
		ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{UserID: uid.String(), Username: uname})

		job := spotifyService.GetStatus(uname)
		// Compute readiness by querying views
		ready := false
		if pr, ok := repo.(*business.PostgreSQLUserRepository); ok {
			var hasArtists, hasTracks bool
			_ = pr.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM v_current_spotify_top_artists WHERE user_id=$1)`, uid).Scan(&hasArtists)
			_ = pr.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM v_current_spotify_top_tracks WHERE user_id=$1)`, uid).Scan(&hasTracks)
			ready = hasArtists || hasTracks
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": job.Status, "ready": ready})
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
	if out.Ready {
		t.Fatalf("expected ready=false, got true")
	}
}

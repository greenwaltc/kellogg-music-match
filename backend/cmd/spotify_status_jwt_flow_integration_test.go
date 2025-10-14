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

// This integration test wires the JWT middleware and ensures the /sync/spotify/status
// endpoint returns ready=false initially and ready=true after inserting a snapshot for the JWT user.
func TestSpotifyStatusJWTFlow(t *testing.T) {
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
	jwtService := business.NewJWTService(&cfg.JWT)
	spotifyService := spotify.NewService(repo, cfg.Spotify.RefreshTokenKey, spotify.WithSpotifyCredentials(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret, cfg.Spotify.RedirectURI))

	// Create a user
	uid := uuid.New()
	suffix := uuid.NewString()[:8]
	uname := fmt.Sprintf("status_jwt_user_%s", suffix)
	email := fmt.Sprintf("jwt_user_%s@example.com", suffix)
	_, err = repo.CreateUser(context.Background(), uid, uname, email, "S", "R", "hash", "2Y", int32(time.Now().Year()+2))
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	// Generate a JWT for that user
	token, err := jwtService.GenerateToken(uid.String(), uname, email)
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	// Build mux with the same status handler logic as main(), protected by JWT middleware
	baseMux := http.NewServeMux()
	baseMux.HandleFunc("/sync/spotify/status", func(w http.ResponseWriter, r *http.Request) {
		// The handler expects the middleware to populate context; reuse main() logic
		uctx, ok := GetUserFromContext(r.Context())
		if !ok || uctx == nil || uctx.UserID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		job := spotifyService.GetStatus(uctx.Username)
		ready := false
		if pr, ok := repo.(*business.PostgreSQLUserRepository); ok {
			var hasArtists, hasTracks bool
			_ = pr.Pool().QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM v_current_spotify_top_artists WHERE user_id=$1)`, uid).Scan(&hasArtists)
			_ = pr.Pool().QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM v_current_spotify_top_tracks WHERE user_id=$1)`, uid).Scan(&hasTracks)
			ready = hasArtists || hasTracks
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": job.Status, "ready": ready})
	})

	// Protect with JWT middleware
	middleware := NewJWTMiddleware(jwtService)
	protected := middleware.Middleware(baseMux)

	// 1) Initially: no snapshots, expect ready=false
	req1 := httptest.NewRequest(http.MethodGet, "/sync/spotify/status", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	protected.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("expected 200, got %d", w1.Code)
	}
	var out1 struct {
		Ready bool `json:"ready"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &out1); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if out1.Ready {
		t.Fatalf("expected ready=false initially, got true")
	}

	// 2) Insert an artist snapshot for the same user, expect ready=true
	fetchedAt := time.Now().UTC()
	if err := repo.StoreSpotifyTopArtists(context.Background(), uid, fetchedAt, "short_term", []business.SpotifyTopArtist{{Rank: 1, SpotifyArtistID: "aid", Name: "Artist"}}); err != nil {
		t.Fatalf("store artist snapshot failed: %v", err)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/sync/spotify/status", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	protected.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var out2 struct {
		Ready bool `json:"ready"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &out2); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if !out2.Ready {
		t.Fatalf("expected ready=true after snapshot, got false")
	}
}

package spotify

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	business "github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
)

// This test expects a running Postgres per docker-compose with env vars configured.
// It will insert real snapshot rows through repository methods.
func TestDBSnapshotPersistence(t *testing.T) {
	if os.Getenv("INTEGRATION_DB") == "" {
		t.Skip("set INTEGRATION_DB=1 to run DB integration test")
	}
	cfg := config.Load()
	repo, err := business.NewUserRepositoryWithConfig(&cfg.Database)
	if err != nil {
		t.Fatalf("repo init: %v", err)
	}
	defer repo.(interface{ Close() error }).Close()

	svc := NewService(repo, cfg.Spotify.RefreshTokenKey, WithTokenWait(5*time.Millisecond, 200*time.Millisecond))
	// Override fetch to deterministic data (avoid external HTTP)
	svc.fetchOverride = func(ctx context.Context, user, rng string) ([]business.SpotifyTopArtist, []business.SpotifyTopTrack, error) {
		artists := []business.SpotifyTopArtist{{Rank: 1, SpotifyArtistID: "artist-int-1", Name: "Integration Artist"}}
		tracks := []business.SpotifyTopTrack{{Rank: 1, SpotifyTrackID: "track-int-1", Name: "Integration Track", ArtistNames: []string{"Integration Artist"}, ArtistIDs: []string{"artist-int-1"}}}
		return artists, tracks, nil
	}

	// Create a temp user (minimal fields)
	uid := uuid.New()
	_, err = repo.CreateUser(context.Background(), uid, "int_user", "int@example.com", "Int", "User", "hash", "2Y", 2026)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Start sync
	svc.StartSync(context.Background(), "int_user", "code", "state")
	svc.SetJobTokens("int_user", uid, "dummy_access", "dummy_refresh", 3600)

	deadline := time.Now().Add(5 * time.Second)
	for {
		st := svc.GetStatus("int_user")
		if st.Status == StatusComplete {
			break
		}
		if st.Status == StatusFailed {
			t.Fatalf("sync failed: %s", st.Message)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting sync completion: %+v", st)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Basic verification: query latest snapshots via direct SQL (simpler than adding new sqlc queries now)
	poolRepo, ok := repo.(*business.PostgreSQLUserRepository)
	if !ok {
		t.Fatalf("unexpected repo concrete type")
	}
	// Count artist snapshots
	var artistCount int
	err = poolRepo.Pool().QueryRow(context.Background(), `SELECT COUNT(*) FROM spotify_top_artist_snapshots WHERE user_id=$1`, uid).Scan(&artistCount)
	if err != nil {
		t.Fatalf("count artist snapshots: %v", err)
	}
	if artistCount == 0 {
		t.Fatalf("expected >0 artist snapshots, got 0")
	}
	var trackCount int
	err = poolRepo.Pool().QueryRow(context.Background(), `SELECT COUNT(*) FROM spotify_top_track_snapshots WHERE user_id=$1`, uid).Scan(&trackCount)
	if err != nil {
		t.Fatalf("count track snapshots: %v", err)
	}
	if trackCount == 0 {
		t.Fatalf("expected >0 track snapshots, got 0")
	}
}

package business_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	business "github.com/greenwaltc/kellogg-music-match/backend/business"
)

// This test exercises the repository methods that page top artists/tracks by delegating to SQL views.
// It attempts a real DB connection; if unavailable, the test is skipped gracefully.
func TestTopItemsRepositoryMethods(t *testing.T) {
	// Attempt to create a real repository (uses env or defaults).
	repoIface, err := business.NewUserRepository()
	if err != nil {
		t.Skipf("skipping: database not available: %v", err)
		return
	}
	// Ensure we close the pool when done if it's a real repo.
	repo, ok := repoIface.(*business.PostgreSQLUserRepository)
	if !ok {
		t.Fatalf("expected PostgreSQLUserRepository, got %T", repoIface)
	}
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.New()
	username := "topitems_user_" + uuid.NewString()[0:8]
	email := fmt.Sprintf("ti+%s@example.com", uuid.NewString()[0:8])

	// Create a user to own the snapshots
	_, err = repo.CreateUser(ctx, userID, username, email, "Top", "Items", "hash", "2Y", 2026)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Insert a small snapshot for artists and tracks
	fetchedAt := time.Now().UTC()
	rng := "medium_term"

	artists := []business.SpotifyTopArtist{
		{Rank: 1, SpotifyArtistID: "a1", Name: "Artist One", Genres: []string{"rock"}},
		{Rank: 2, SpotifyArtistID: "a2", Name: "Artist Two", Genres: []string{"pop"}},
	}
	if err := repo.StoreSpotifyTopArtists(ctx, userID, fetchedAt, rng, artists); err != nil {
		t.Fatalf("StoreSpotifyTopArtists failed: %v", err)
	}

	tracks := []business.SpotifyTopTrack{
		{Rank: 1, SpotifyTrackID: "t1", Name: "Track One", ArtistNames: []string{"Artist One"}},
		{Rank: 2, SpotifyTrackID: "t2", Name: "Track Two", ArtistNames: []string{"Artist Two"}},
	}
	if err := repo.StoreSpotifyTopTracks(ctx, userID, fetchedAt, rng, tracks); err != nil {
		t.Fatalf("StoreSpotifyTopTracks failed: %v", err)
	}

	// Artists: page size 1, offset 0 -> should return rank 1
	gotArtists, err := repo.GetUserTopArtistsByRange(ctx, userID, rng, 1, 0)
	if err != nil {
		t.Fatalf("GetUserTopArtistsByRange err: %v", err)
	}
	if len(gotArtists) != 1 || gotArtists[0].Rank != 1 || gotArtists[0].SpotifyArtistID != "a1" {
		t.Fatalf("unexpected artists page0: %+v", gotArtists)
	}
	// Artists: page size 1, offset 1 -> should return rank 2
	gotArtists, err = repo.GetUserTopArtistsByRange(ctx, userID, rng, 1, 1)
	if err != nil {
		t.Fatalf("GetUserTopArtistsByRange(off=1) err: %v", err)
	}
	if len(gotArtists) != 1 || gotArtists[0].Rank != 2 || gotArtists[0].SpotifyArtistID != "a2" {
		t.Fatalf("unexpected artists page1: %+v", gotArtists)
	}

	// Tracks similar checks
	gotTracks, err := repo.GetUserTopTracksByRange(ctx, userID, rng, 1, 0)
	if err != nil {
		t.Fatalf("GetUserTopTracksByRange err: %v", err)
	}
	if len(gotTracks) != 1 || gotTracks[0].Rank != 1 || gotTracks[0].SpotifyTrackID != "t1" {
		t.Fatalf("unexpected tracks page0: %+v", gotTracks)
	}
	gotTracks, err = repo.GetUserTopTracksByRange(ctx, userID, rng, 1, 1)
	if err != nil {
		t.Fatalf("GetUserTopTracksByRange(off=1) err: %v", err)
	}
	if len(gotTracks) != 1 || gotTracks[0].Rank != 2 || gotTracks[0].SpotifyTrackID != "t2" {
		t.Fatalf("unexpected tracks page1: %+v", gotTracks)
	}
}

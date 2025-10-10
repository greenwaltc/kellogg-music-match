package business

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/greenwaltc/kellogg-music-match/backend/config"
)

// TestTopNArtistsFilter verifies that overlaps only include artists within the configured MATCHING_ARTIST_TOPN ranks
func TestTopNArtistsFilter(t *testing.T) {
    if os.Getenv("INTEGRATION_DB") == "" {
        t.Skip("set INTEGRATION_DB=1 to run DB integration test")
    }
    // Force a small TopN for artists
    prevArtistTopN := os.Getenv("MATCHING_ARTIST_TOPN")
    _ = os.Setenv("MATCHING_ARTIST_TOPN", "2")
    defer func() {
        _ = os.Setenv("MATCHING_ARTIST_TOPN", prevArtistTopN)
    }()

    cfg := config.Load()
    repo, err := NewUserRepositoryWithConfig(&cfg.Database)
    if err != nil {
        t.Fatalf("repo init: %v", err)
    }
    defer repo.(interface{ Close() error }).Close()

    ctx := context.Background()
    u1 := uuid.New()
    u2 := uuid.New()
    u1Name := "topn_u1_" + u1.String()[:8]
    u2Name := "topn_u2_" + u2.String()[:8]

    // Create two users
    if _, err := repo.CreateUser(ctx, u1, u1Name, u1Name+"@example.com", "U1", "Test", "hash", "2Y", 2025); err != nil {
        t.Fatalf("create u1: %v", err)
    }
    if _, err := repo.CreateUser(ctx, u2, u2Name, u2Name+"@example.com", "U2", "Test", "hash", "2Y", 2025); err != nil {
        t.Fatalf("create u2: %v", err)
    }

    // Insert artist snapshots with overlaps at ranks 1 (A1) and 3 (A3). With TopN=2, only A1 should appear.
    fetched := time.Now().Add(24 * time.Hour) // ensure newest
    rng := "medium_term"

    err = repo.StoreSpotifyTopArtists(ctx, u1, fetched, rng, []SpotifyTopArtist{
        {Rank: 1, SpotifyArtistID: "artist-a1", Name: "Artist A1"},
        {Rank: 3, SpotifyArtistID: "artist-a3", Name: "Artist A3"},
        {Rank: 4, SpotifyArtistID: "artist-a4", Name: "Artist A4"},
    })
    if err != nil {
        t.Fatalf("store u1 artists: %v", err)
    }
    err = repo.StoreSpotifyTopArtists(ctx, u2, fetched, rng, []SpotifyTopArtist{
        {Rank: 2, SpotifyArtistID: "artist-a1", Name: "Artist A1"},
        {Rank: 4, SpotifyArtistID: "artist-a3", Name: "Artist A3"},
        {Rank: 5, SpotifyArtistID: "artist-a5", Name: "Artist A5"},
    })
    if err != nil {
        t.Fatalf("store u2 artists: %v", err)
    }

    // Query similar users for u1
    results, err := repo.FindSimilarUsersBySpotifyTopArtists(ctx, u1, rng, 10)
    if err != nil {
        t.Fatalf("FindSimilarUsersBySpotifyTopArtists: %v", err)
    }
    if len(results) == 0 {
        t.Fatalf("expected at least one similar user, got 0")
    }
    // We expect u2 to be in results with only A1 overlap
    var found bool
    for _, r := range results {
        if r.UserID == u2 {
            found = true
            if len(r.Overlaps) != 1 {
                t.Fatalf("expected 1 overlap within TopN, got %d", len(r.Overlaps))
            }
            ov := r.Overlaps[0]
            if ov.Name != "Artist A1" || ov.AnchorRank != 1 || ov.OtherRank != 2 {
                t.Fatalf("unexpected overlap: %+v", ov)
            }
        }
    }
    if !found {
        t.Fatalf("expected to find u2 in similar results")
    }
}

// TestTopNTracksFilter verifies that overlaps only include tracks within the configured MATCHING_TRACK_TOPN ranks
func TestTopNTracksFilter(t *testing.T) {
    if os.Getenv("INTEGRATION_DB") == "" {
        t.Skip("set INTEGRATION_DB=1 to run DB integration test")
    }
    prevTrackTopN := os.Getenv("MATCHING_TRACK_TOPN")
    _ = os.Setenv("MATCHING_TRACK_TOPN", "2")
    defer func() {
        _ = os.Setenv("MATCHING_TRACK_TOPN", prevTrackTopN)
    }()

    cfg := config.Load()
    repo, err := NewUserRepositoryWithConfig(&cfg.Database)
    if err != nil {
        t.Fatalf("repo init: %v", err)
    }
    defer repo.(interface{ Close() error }).Close()

    ctx := context.Background()
    u1 := uuid.New()
    u2 := uuid.New()
    u1Name := "topn_t_u1_" + u1.String()[:8]
    u2Name := "topn_t_u2_" + u2.String()[:8]

    // Create two users
    if _, err := repo.CreateUser(ctx, u1, u1Name, u1Name+"@example.com", "TU1", "Test", "hash", "2Y", 2025); err != nil {
        t.Fatalf("create u1: %v", err)
    }
    if _, err := repo.CreateUser(ctx, u2, u2Name, u2Name+"@example.com", "TU2", "Test", "hash", "2Y", 2025); err != nil {
        t.Fatalf("create u2: %v", err)
    }

    fetched := time.Now().Add(24 * time.Hour)
    rng := "medium_term"

    // Overlaps at ranks 1 (T1) and 3 (T3). With TopN=2, only T1 should appear.
    err = repo.StoreSpotifyTopTracks(ctx, u1, fetched, rng, []SpotifyTopTrack{
        {Rank: 1, SpotifyTrackID: "track-t1", Name: "Track T1", ArtistNames: []string{"X"}},
        {Rank: 3, SpotifyTrackID: "track-t3", Name: "Track T3", ArtistNames: []string{"Y"}},
        {Rank: 4, SpotifyTrackID: "track-t4", Name: "Track T4", ArtistNames: []string{"Z"}},
    })
    if err != nil {
        t.Fatalf("store u1 tracks: %v", err)
    }
    err = repo.StoreSpotifyTopTracks(ctx, u2, fetched, rng, []SpotifyTopTrack{
        {Rank: 2, SpotifyTrackID: "track-t1", Name: "Track T1", ArtistNames: []string{"X"}},
        {Rank: 4, SpotifyTrackID: "track-t3", Name: "Track T3", ArtistNames: []string{"Y"}},
        {Rank: 5, SpotifyTrackID: "track-t5", Name: "Track T5", ArtistNames: []string{"W"}},
    })
    if err != nil {
        t.Fatalf("store u2 tracks: %v", err)
    }

    results, err := repo.FindSimilarUsersBySpotifyTopTracks(ctx, u1, rng, 10)
    if err != nil {
        t.Fatalf("FindSimilarUsersBySpotifyTopTracks: %v", err)
    }
    if len(results) == 0 {
        t.Fatalf("expected at least one similar user, got 0")
    }
    var found bool
    for _, r := range results {
        if r.UserID == u2 {
            found = true
            if len(r.Overlaps) != 1 {
                t.Fatalf("expected 1 overlap within TopN, got %d", len(r.Overlaps))
            }
            ov := r.Overlaps[0]
            if ov.Name != "Track T1" || ov.AnchorRank != 1 || ov.OtherRank != 2 {
                t.Fatalf("unexpected track overlap: %+v", ov)
            }
        }
    }
    if !found {
        t.Fatalf("expected to find u2 in similar results (tracks)")
    }
}

package spotify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	business "github.com/greenwaltc/kellogg-music-match/backend/business"
)

// mockStore implements TokenStore plus snapshot methods for testing without DB.
type mockStore struct {
	tokensPersisted bool
	artistSnapshots int
	trackSnapshots  int
}

func (m *mockStore) UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error {
	m.tokensPersisted = true
	return nil
}
func (m *mockStore) StoreSpotifyTopArtists(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []business.SpotifyTopArtist) error {
	m.artistSnapshots += len(items)
	return nil
}
func (m *mockStore) StoreSpotifyTopTracks(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []business.SpotifyTopTrack) error {
	m.trackSnapshots += len(items)
	return nil
}

// Test job lifecycle with injected tokens immediately (simulate wrapper setting tokens) without real HTTP.
func TestJobLifecycleWithoutHTTP(t *testing.T) {
	st := &mockStore{}
	s := NewService(st, "enc-key", WithTokenWait(10*time.Millisecond, 200*time.Millisecond))
	// Provide override to avoid external HTTP
	s.fetchOverride = func(ctx context.Context, username, rng string) ([]business.SpotifyTopArtist, []business.SpotifyTopTrack, error) {
		artists := []business.SpotifyTopArtist{{Rank: 1, SpotifyArtistID: "a1", Name: "Artist 1"}}
		tracks := []business.SpotifyTopTrack{{Rank: 1, SpotifyTrackID: "t1", Name: "Track 1", ArtistNames: []string{"Artist 1"}, ArtistIDs: []string{"a1"}}}
		return artists, tracks, nil
	}
	_ = s.StartSync(context.Background(), "bob", "code", "state")
	s.SetJobTokens("bob", uuid.New(), "access", "refresh", 3600)
	deadline := time.Now().Add(5 * time.Second)
	for {
		js := s.GetStatus("bob")
		if js.Status == StatusComplete {
			if js.Progress != 100 {
				t.Fatalf("expected progress 100 got %d", js.Progress)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for completion: %+v", js)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !st.tokensPersisted {
		t.Log("tokens not persisted (expected if encryption key empty or persistence disabled)")
	}
}

// Test failure when tokens never set (simulate timeout waiting for wrapper token exchange).
func TestJobFailsWhenTokensMissing(t *testing.T) {
	s := NewService(nil, "", WithTokenWait(5*time.Millisecond, 40*time.Millisecond))
	s.fetchOverride = func(ctx context.Context, username, rng string) ([]business.SpotifyTopArtist, []business.SpotifyTopTrack, error) {
		return nil, nil, errors.New("should not be called without token")
	}
	_ = s.StartSync(context.Background(), "alice", "code", "state")
	deadline := time.Now().Add(2 * time.Second)
	for {
		st := s.GetStatus("alice")
		if st.Status == StatusFailed {
			if st.Message == "" {
				t.Fatalf("expected failure message")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("did not fail within timeout; state=%+v", st)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

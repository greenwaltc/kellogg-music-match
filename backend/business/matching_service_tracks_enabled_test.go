package business

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// mock repo to assert track path invoked when flag enabled
type mockUserRepoTracks struct {
	UserRepository
	artistsCalled int
	tracksCalled  int
	user          *sqlc.User
}

func (m *mockUserRepoTracks) GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error) {
	return m.user, nil
}
func (m *mockUserRepoTracks) FindSimilarUsersBySpotifyTopArtists(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]SimilarUserResult, error) {
	m.artistsCalled++
	return nil, nil
}
func (m *mockUserRepoTracks) FindSimilarUsersBySpotifyTopArtistsFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string) ([]SimilarUserResult, error) {
	m.artistsCalled++
	return nil, nil
}
func (m *mockUserRepoTracks) FindSimilarUsersBySpotifyTopTracks(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]SimilarUserResult, error) {
	m.tracksCalled++
	return nil, nil
}
func (m *mockUserRepoTracks) FindSimilarUsersBySpotifyTopTracksFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string) ([]SimilarUserResult, error) {
	m.tracksCalled++
	return nil, nil
}

func TestMatchingService_BasisTracksEnabled(t *testing.T) {
	// Enable tracks via env var
	os.Setenv("MATCH_TRACKS_ENABLED", "true")
	// Reset config by reloading inside service construction (Load reads env each time)
	repo := &mockUserRepoTracks{user: &sqlc.User{ID: uuid.New(), Username: "bob"}}
	ms := NewMatchingService(repo, NewMatchingEngine())
	ctx := context.WithValue(context.Background(), MatchBasisContextKey{}, "tracks")
	imp, err := ms.FindMusicMatches(ctx, generated.ArtistsRequest{}, "bob", "medium_term", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imp.Code != 200 {
		t.Fatalf("expected 200 got %d", imp.Code)
	}
	if repo.tracksCalled != 1 {
		t.Fatalf("expected tracks path invoked once, got %d (artists %d)", repo.tracksCalled, repo.artistsCalled)
	}
}

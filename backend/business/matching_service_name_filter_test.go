package business

import (
	"context"
	"testing"

	"github.com/google/uuid"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// mock repo to assert filtered vs unfiltered path
type mockUserRepoNameFilter struct {
	UserRepository
	user                  *sqlc.User
	artistsCalled         int
	artistsFilteredCalled int
	tracksCalled          int
	tracksFilteredCalled  int
}

func (m *mockUserRepoNameFilter) GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error) {
	return m.user, nil
}
func (m *mockUserRepoNameFilter) FindSimilarUsersBySpotifyTopArtists(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, includeDetails bool) ([]SimilarUserResult, error) {
	m.artistsCalled++
	return []SimilarUserResult{}, nil
}
func (m *mockUserRepoNameFilter) FindSimilarUsersBySpotifyTopArtistsFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string, includeDetails bool) ([]SimilarUserResult, error) {
	m.artistsFilteredCalled++
	return []SimilarUserResult{}, nil
}
func (m *mockUserRepoNameFilter) FindSimilarUsersBySpotifyTopTracks(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, includeDetails bool) ([]SimilarUserResult, error) {
	m.tracksCalled++
	return []SimilarUserResult{}, nil
}
func (m *mockUserRepoNameFilter) FindSimilarUsersBySpotifyTopTracksFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string, includeDetails bool) ([]SimilarUserResult, error) {
	m.tracksFilteredCalled++
	return []SimilarUserResult{}, nil
}

func TestMatchingService_NameFilter_ArtistsPath(t *testing.T) {
	repo := &mockUserRepoNameFilter{user: &sqlc.User{ID: uuid.New(), Username: "alice"}}
	ms := NewMatchingService(repo, NewMatchingEngine())
	// No filter -> unfiltered artists
	_, err := ms.FindMusicMatches(context.Background(), generated.ArtistsRequest{}, "alice", "medium_term", true, 10)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.artistsCalled != 1 || repo.artistsFilteredCalled != 0 {
		t.Fatalf("expected unfiltered artists; got artists=%d filtered=%d", repo.artistsCalled, repo.artistsFilteredCalled)
	}
	// With filter -> filtered artists
	ctx := context.WithValue(context.Background(), MatchNameFilterContextKey{}, "First Last")
	_, err = ms.FindMusicMatches(ctx, generated.ArtistsRequest{}, "alice", "medium_term", true, 10)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.artistsFilteredCalled != 1 {
		t.Fatalf("expected filtered artists path once; got %d", repo.artistsFilteredCalled)
	}
}

func TestMatchingService_NameFilter_TracksPath(t *testing.T) {
	// Enable tracks by setting basis in context
	repo := &mockUserRepoNameFilter{user: &sqlc.User{ID: uuid.New(), Username: "alice"}}
	ms := NewMatchingService(repo, NewMatchingEngine())
	// Tracks disabled by config default -> expect 400 and no repo calls
	ctx := context.WithValue(context.Background(), MatchBasisContextKey{}, "tracks")
	imp, err := ms.FindMusicMatches(ctx, generated.ArtistsRequest{}, "alice", "medium_term", true, 10)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if imp.Code != 400 {
		t.Fatalf("expected 400 when tracks disabled; got %d", imp.Code)
	}
	if repo.tracksCalled != 0 && repo.tracksFilteredCalled != 0 {
		t.Fatalf("expected no tracks calls when disabled")
	}
}

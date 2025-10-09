package business

import (
	"context"
	"testing"

	"github.com/google/uuid"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// mock repo implementing UserRepository subset for basis tests
type mockUserRepoBasis struct {
	UserRepository
	artistsCalled int
	tracksCalled  int
	tracksErr     error
	tracksResults []SimilarUserResult
	artistResults []SimilarUserResult
	user          *sqlc.User
}

func (m *mockUserRepoBasis) GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error) {
	if m.user == nil {
		return nil, nil
	}
	return m.user, nil
}
func (m *mockUserRepoBasis) FindSimilarUsersBySpotifyTopArtists(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]SimilarUserResult, error) {
	m.artistsCalled++
	return m.artistResults, nil
}
func (m *mockUserRepoBasis) FindSimilarUsersBySpotifyTopTracks(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]SimilarUserResult, error) {
	m.tracksCalled++
	if m.tracksErr != nil {
		return nil, m.tracksErr
	}
	return m.tracksResults, nil
}

func TestMatchingService_BasisArtistsDefault(t *testing.T) {
	repo := &mockUserRepoBasis{user: &sqlc.User{ID: uuid.New(), Username: "alice"}}
	ms := NewMatchingService(repo, NewMatchingEngine())
	resp, err := ms.FindMusicMatches(context.Background(), generated.ArtistsRequest{}, "alice", "medium_term", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.artistsCalled != 1 || repo.tracksCalled != 0 {
		t.Errorf("expected artists path (1) tracks (0) got %d %d", repo.artistsCalled, repo.tracksCalled)
	}
	_ = resp
}

func TestMatchingService_BasisTracksFlagDisabled(t *testing.T) {
	repo := &mockUserRepoBasis{user: &sqlc.User{ID: uuid.New(), Username: "alice"}}
	ms := NewMatchingService(repo, NewMatchingEngine())
	ctx := context.WithValue(context.Background(), "match_basis", "tracks")
	// Feature flag currently default false, expect 400 status
	imp, err := ms.FindMusicMatches(ctx, generated.ArtistsRequest{}, "alice", "medium_term", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.tracksCalled != 0 {
		t.Errorf("expected tracks not invoked when flag disabled")
	}
	if imp.Code != 400 {
		t.Errorf("expected 400 code got %d", imp.Code)
	}
}

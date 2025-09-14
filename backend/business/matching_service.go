package business

import (
	"context"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// MatchingService implements the business logic for music matching
type MatchingService struct {
	store    *Store
	matching *MatchingEngine
}

// NewMatchingService creates a new matching service
func NewMatchingService(store *Store, matching *MatchingEngine) *MatchingService {
	return &MatchingService{
		store:    store,
		matching: matching,
	}
}

// FindMusicMatches implements music matching business logic
func (s *MatchingService) FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, xUserUsername string) (generated.ImplResponse, error) {
	// Validate input
	if xUserUsername == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "username header is required",
		}), nil
	}

	if len(artistsRequest.Artists) == 0 {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "at least one artist is required",
		}), nil
	}

	// Get all users from store
	users := s.store.SnapshotUsers()

	// Find matches using the algorithm
	matches := s.matching.ComputeMatches(artistsRequest.Artists, xUserUsername, users)

	return generated.Response(http.StatusOK, matches), nil
}

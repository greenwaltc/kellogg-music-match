package business

import (
	"context"
	"fmt"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// toStringSlice converts various possible PostgreSQL array representations into a []string.
// toStringSlice removed with legacy similarity system.

// MatchingService implements the business logic for music matching
type MatchingService struct {
	userRepo     UserRepository
	matching     *MatchingEngine
	artistConfig *config.ArtistConfig
}

// NewMatchingService creates a new matching service with default config
func NewMatchingService(userRepo UserRepository, matching *MatchingEngine) *MatchingService {
	cfg := config.Load()
	return &MatchingService{
		userRepo:     userRepo,
		matching:     matching,
		artistConfig: &cfg.Artist,
	}
}

// NewMatchingServiceWithConfig creates a new matching service with provided config
func NewMatchingServiceWithConfig(userRepo UserRepository, matching *MatchingEngine, artistConfig *config.ArtistConfig) *MatchingService {
	return &MatchingService{
		userRepo:     userRepo,
		matching:     matching,
		artistConfig: artistConfig,
	}
}

// FindMusicMatches implements music matching business logic
func (s *MatchingService) FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, xUserUsername string) (generated.ImplResponse, error) {
	fmt.Printf("DEBUG: FindMusicMatches (Spotify placeholder) user=%s ignoring manual artists payload=%v\n", xUserUsername, artistsRequest.Artists)

	if xUserUsername == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "username header is required"}), nil
	}

	// Placeholder: no similarity computation until Spotify preference ingestion implemented.
	matches := []*generated.MatchUser{}
	// Keep the humorous entry to preserve UI expectations if any.
	crushMatch := &generated.MatchUser{
		Name:           "Your Kellogg MBA Crush",
		Program:        "2Y",
		GraduationYear: 2026,
		Overlap:        0,
		Score:          0,
		Artists:        []string{"Obscure artist 1", "Obscure artist 2", "Obscure artist 3", "Obscure artist 4"},
	}
	matches = append(matches, crushMatch)
	return generated.Response(http.StatusOK, matches), nil
}

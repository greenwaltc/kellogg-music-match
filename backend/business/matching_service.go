package business

import (
	"context"
	"fmt"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// MatchingService implements the business logic for music matching
type MatchingService struct {
	userRepo UserRepository
	matching *MatchingEngine
}

// NewMatchingService creates a new matching service
func NewMatchingService(userRepo UserRepository, matching *MatchingEngine) *MatchingService {
	return &MatchingService{
		userRepo: userRepo,
		matching: matching,
	}
}

// FindMusicMatches implements music matching business logic
func (s *MatchingService) FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, xUserUsername string) (generated.ImplResponse, error) {
	fmt.Printf("DEBUG: FindMusicMatches called for user: %s with artists: %v\n", xUserUsername, artistsRequest.Artists)

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

	// If username is provided, update user's artist preferences
	if xUserUsername != "" {
		// First get the user to check if they exist
		user, err := s.userRepo.GetUserByUsername(ctx, xUserUsername)
		if err != nil {
			return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
				Message: "failed to retrieve user",
			}), nil
		}

		if user == nil {
			return generated.Response(http.StatusNotFound, generated.ErrorResponse{
				Message: "user not found",
			}), nil
		}

		// Update user's artists
		fmt.Printf("DEBUG: Attempting to set artists for user ID: %s, artists: %v\n", user.ID.String(), artistsRequest.Artists)
		err = s.userRepo.SetUserArtists(ctx, user.ID, artistsRequest.Artists)
		if err != nil {
			fmt.Printf("ERROR: SetUserArtists failed: %v\n", err)
			return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
				Message: "failed to update user artists",
			}), nil
		}
		fmt.Printf("DEBUG: Successfully set artists for user: %s\n", user.Username)
	}

	similarUsers, err := s.userRepo.FindSimilarUsers(ctx, xUserUsername)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to query similar users",
		}), nil
	}

	// Debug: log similar users found
	fmt.Printf("FindSimilarUsers returned %d similar users for %s\n", len(similarUsers), xUserUsername)
	for _, row := range similarUsers {
		// Convert artists interface{} to []string for logging
		var artistNames []string
		if row.Artists != nil {
			if names, ok := row.Artists.([]interface{}); ok {
				artistNames = make([]string, 0, len(names))
				for _, name := range names {
					if str, ok := name.(string); ok {
						artistNames = append(artistNames, str)
					}
				}
			}
		}
		fmt.Printf("Similar user: %s (distance: %.3f) with artists: %v\n", row.Username, row.Distance, artistNames)
	}

	// Convert similar users to API response format
	matches := make([]*generated.MatchUser, 0, len(similarUsers))
	for _, row := range similarUsers {
		// Convert artists interface{} to []string
		var artistNames []string
		if row.Artists != nil {
			if names, ok := row.Artists.([]interface{}); ok {
				artistNames = make([]string, 0, len(names))
				for _, name := range names {
					if str, ok := name.(string); ok {
						artistNames = append(artistNames, str)
					}
				}
			}
		}

		// Calculate overlap count (could be enhanced to compare with user's artists)
		overlapCount := len(artistNames) // Simple count for now

		match := &generated.MatchUser{
			Name:    fmt.Sprintf("%s %s", row.FirstName, row.LastName),
			Overlap: int32(overlapCount),
			Score:   float32(1.0 - row.Distance), // Convert distance to similarity score (higher is better)
		}
		matches = append(matches, match)
	}

	return generated.Response(http.StatusOK, matches), nil
}

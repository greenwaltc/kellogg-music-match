package business

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

	// Convert current user's artists to a map for efficient lookup
	userArtistsMap := make(map[string]bool)
	for _, artist := range artistsRequest.Artists {
		userArtistsMap[strings.ToLower(strings.TrimSpace(artist))] = true
	}

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

		// Calculate actual overlap count between user's artists and matched user's artists
		overlapCount := 0
		for _, artist := range artistNames {
			normalizedArtist := strings.ToLower(strings.TrimSpace(artist))
			if userArtistsMap[normalizedArtist] {
				overlapCount++
			}
		}

		match := &generated.MatchUser{
			Name:    fmt.Sprintf("%s %s", row.FirstName, row.LastName),
			Overlap: int32(overlapCount),
			Score:   float32(1.0 - row.Distance), // Convert distance to similarity score (higher is better)
			Artists: artistNames,                 // Include the artists array
		}
		matches = append(matches, match)
	}

	// Add the funny "Kellogg MBA Crush" component only when matches are returned
	if len(matches) > 0 {
		crushMatch := &generated.MatchUser{
			Name:    "Your Kellogg MBA Crush",
			Overlap: int32(0),                                                     // No overlap by design
			Score:   float32(0.0),                                                 // No compatibility
			Artists: []string{"Classical Music", "Podcasts", "NPR", "True Crime"}, // Completely different taste
		}
		matches = append(matches, crushMatch)
	}

	return generated.Response(http.StatusOK, matches), nil
}

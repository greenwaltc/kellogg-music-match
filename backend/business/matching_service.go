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
		err = s.userRepo.SetUserArtists(ctx, user.ID, artistsRequest.Artists)
		if err != nil {
			return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
				Message: "failed to update user artists",
			}), nil
		}
	}

	// Get all users with their artists from database
	dbUsersWithArtists, err := s.userRepo.GetAllUsersWithArtists(ctx)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to retrieve users",
		}), nil
	}

	// Debug: log the number of users retrieved
	fmt.Printf("GetAllUsersWithArtists returned %d users\n", len(dbUsersWithArtists))

	// Convert database users to API users
	users := make([]*generated.User, 0, len(dbUsersWithArtists))

	for _, row := range dbUsersWithArtists {
		// Convert artist_names interface{} to []string
		var artistNames []string
		if row.ArtistNames != nil {
			if names, ok := row.ArtistNames.([]interface{}); ok {
				artistNames = make([]string, 0, len(names))
				for _, name := range names {
					if str, ok := name.(string); ok {
						artistNames = append(artistNames, str)
					}
				}
			}
		}

		user := &generated.User{
			Id:        row.ID.String(),
			Username:  row.Username,
			Email:     row.Email,
			FirstName: row.FirstName,
			LastName:  row.LastName,
			Artists:   artistNames,
		}
		users = append(users, user)
	}

	// Find matches using the algorithm
	matches := s.matching.ComputeMatches(artistsRequest.Artists, xUserUsername, users)
	
	// Debug: log the matches
	fmt.Printf("ComputeMatches returned %d matches for user %s\n", len(matches), xUserUsername)

	return generated.Response(http.StatusOK, matches), nil
}

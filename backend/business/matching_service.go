package business

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// SearchArtists implements fuzzy artist search functionality
func (s *MatchingService) SearchArtists(ctx context.Context, query string, limit int32) (generated.ImplResponse, error) {
	// Validate input
	if strings.TrimSpace(query) == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "search query is required",
		}), nil
	}

	if len(query) > 240 {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "search query must be 240 characters or less",
		}), nil
	}

	// Set default limit if not provided
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Perform fuzzy search using the repository
	artists, err := s.userRepo.SearchArtists(ctx, query, limit)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to search artists",
		}), nil
	}

	// Convert to API response format
	apiArtists := make([]generated.Artist, 0, len(artists))
	for _, artist := range artists {
		createdAt := artist.CreatedAt.Time
		if !artist.CreatedAt.Valid {
			createdAt = time.Time{} // Use zero time if null
		}
		apiArtists = append(apiArtists, generated.Artist{
			Id:        int32(artist.ID),
			Name:      artist.Name,
			CreatedAt: createdAt,
		})
	}

	return generated.Response(http.StatusOK, apiArtists), nil
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

	// Validate artist name lengths
	for _, artist := range artistsRequest.Artists {
		if len(artist) > 240 {
			return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
				Message: "artist names must be 240 characters or less",
			}), nil
		}
	}

	// Check for duplicate artists (case-insensitive)
	artistSet := make(map[string]bool)
	for _, artist := range artistsRequest.Artists {
		artistLower := strings.ToLower(strings.TrimSpace(artist))
		if artistLower == "" {
			continue // Skip empty artists
		}
		fmt.Printf("DEBUG: Checking artist: '%s' -> '%s'\n", artist, artistLower)
		if artistSet[artistLower] {
			fmt.Printf("DEBUG: Found duplicate: '%s'\n", artistLower)
			return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
				Message: "duplicate artists are not allowed",
			}), nil
		}
		artistSet[artistLower] = true
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
			// Handle different possible types for PostgreSQL arrays
			switch v := row.Artists.(type) {
			case []interface{}:
				artistNames = make([]string, 0, len(v))
				for _, name := range v {
					if str, ok := name.(string); ok {
						artistNames = append(artistNames, str)
					}
				}
			case []string:
				artistNames = v
			case []uint8:
				// PostgreSQL array as byte array representing JSON - convert to string first
				jsonStr := string(v)
				if len(jsonStr) > 2 && jsonStr[0] == '{' && jsonStr[len(jsonStr)-1] == '}' {
					jsonStr = jsonStr[1 : len(jsonStr)-1] // Remove { and }
					if jsonStr != "" {
						// Split by comma and handle quoted values
						parts := strings.Split(jsonStr, ",")
						for _, part := range parts {
							part = strings.TrimSpace(part)
							part = strings.Trim(part, `"`) // Remove quotes
							if part != "" {
								artistNames = append(artistNames, part)
							}
						}
					}
				}
			case string:
				// Sometimes PostgreSQL returns arrays as strings
				// Remove curly braces and split by comma
				if len(v) > 2 && v[0] == '{' && v[len(v)-1] == '}' {
					v = v[1 : len(v)-1] // Remove { and }
					if v != "" {
						artistNames = strings.Split(v, ",")
						// Trim whitespace and quotes from each artist name
						for i, artist := range artistNames {
							artist = strings.Trim(artist, ` "`)
							artistNames[i] = artist
						}
					}
				}
			default:
				// Fallback - try to log for any remaining edge cases but don't fail
				fmt.Printf("DEBUG: Unhandled artists type: %T\n", v)
			}
		}

		// Calculate actual overlap count between user's artists and matched user's artists
		overlapCount := 0
		for _, artist := range artistNames {
			normalizedArtist := strings.ToLower(strings.TrimSpace(artist))
			if userArtistsMap[normalizedArtist] {
				overlapCount++
				fmt.Printf("DEBUG: Found overlap: %s\n", artist)
			}
		}
		fmt.Printf("DEBUG: User %s - overlap: %d, distance: %.3f, artists: %v\n", row.Username, overlapCount, row.Distance, artistNames)

		// Use the enhanced hybrid similarity score directly from PostgreSQL
		// The spearman_distance function now includes:
		// - Jaccard similarity (intersection/union)
		// - Positional correlation for shared items
		// - Size penalty for variable-length lists
		// Distance ranges from 0 (identical) to 2 (completely different)
		
		// Convert distance to similarity score (0-1 range, higher is better)
		score := float32(1.0 - (row.Distance / 2.0))
		
		fmt.Printf("DEBUG: User %s - hybrid distance: %.3f, similarity score: %.3f\n", row.Username, row.Distance, score)

		match := &generated.MatchUser{
			Name:           fmt.Sprintf("%s %s", row.FirstName, row.LastName),
			Program:        row.Program.String,
			GraduationYear: row.GraduationYear.Int32,
			Overlap:        int32(overlapCount),
			Score:          score,
			Artists:        artistNames,
		}
		matches = append(matches, match)
	}

	// Add the funny "Kellogg MBA Crush" component only when matches are returned
	if len(matches) > 0 {
		crushMatch := &generated.MatchUser{
			Name:           "Your Kellogg MBA Crush",
			Program:        "2Y",                                                         // Default program for the joke entry
			GraduationYear: 2026,                                                         // Default graduation year for the joke entry
			Overlap:        int32(0),                                                     // No overlap by design
			Score:          float32(0.0),                                                 // No compatibility
			Artists:        []string{"Classical Music", "Podcasts", "NPR", "True Crime"}, // Completely different taste
		}
		matches = append(matches, crushMatch)
	}

	return generated.Response(http.StatusOK, matches), nil
}

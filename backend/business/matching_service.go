package business

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// toStringSlice converts various possible PostgreSQL array representations into a []string.
func toStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []string:
		return t
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []uint8:
		// e.g., "{\"A\",\"B\"}" or "{A,B}" representations
		s := string(t)
		if len(s) > 1 && s[0] == '{' && s[len(s)-1] == '}' {
			s = s[1 : len(s)-1]
			if s == "" {
				return nil
			}
			parts := strings.Split(s, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.Trim(p, `"`)
				if p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		return nil
	case string:
		// e.g., "{A,B}" format
		s := t
		if len(s) > 1 && s[0] == '{' && s[len(s)-1] == '}' {
			s = s[1 : len(s)-1]
			if s == "" {
				return nil
			}
			parts := strings.Split(s, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.Trim(p, `"`)
				if p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		return []string{s}
	default:
		return nil
	}
}

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

// SearchArtists implements fuzzy artist search functionality
func (s *MatchingService) SearchArtists(ctx context.Context, query string, limit int32) (generated.ImplResponse, error) {
	// Validate input
	if strings.TrimSpace(query) == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "search query is required",
		}), nil
	}

	if len(query) > s.artistConfig.SearchMaxLength {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: fmt.Sprintf("search query must be %d characters or less", s.artistConfig.SearchMaxLength),
		}), nil
	}

	// Set default limit if not provided
	if limit <= 0 {
		limit = int32(s.artistConfig.SearchLimit)
	}
	if limit > int32(s.artistConfig.SearchLimit) {
		limit = int32(s.artistConfig.SearchLimit)
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
			Message: fmt.Sprintf("at least %d artists are required", s.artistConfig.MinCount),
		}), nil
	}

	if len(artistsRequest.Artists) < s.artistConfig.MinCount {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: fmt.Sprintf("at least %d artists are required", s.artistConfig.MinCount),
		}), nil
	}

	if len(artistsRequest.Artists) > s.artistConfig.MaxCount {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: fmt.Sprintf("maximum of %d artists allowed", s.artistConfig.MaxCount),
		}), nil
	}

	// Validate artist name lengths
	for _, artist := range artistsRequest.Artists {
		if len(artist) > s.artistConfig.MaxNameLength {
			return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
				Message: fmt.Sprintf("artist names must be %d characters or less", s.artistConfig.MaxNameLength),
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

	// Validate we have enough valid artists after filtering
	validArtistCount := len(artistSet)
	if validArtistCount < s.artistConfig.MinCount {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: fmt.Sprintf("at least %d valid artists are required", s.artistConfig.MinCount),
		}), nil
	}
	if validArtistCount > s.artistConfig.MaxCount {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: fmt.Sprintf("maximum of %d artists allowed", s.artistConfig.MaxCount),
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
		fmt.Printf("Similar user: %s (distance: %.3f) with artists: %v\n", row.Username, row.Distance, toStringSlice(row.Artists))
	}

	// Convert similar users to API response format
	matches := make([]*generated.MatchUser, 0, len(similarUsers))

	// Convert current user's artists to a map for efficient lookup
	userArtistsMap := make(map[string]bool)
	for _, artist := range artistsRequest.Artists {
		userArtistsMap[strings.ToLower(strings.TrimSpace(artist))] = true
	}

	for _, row := range similarUsers {
		// Convert artists to []string for downstream use
		artistNames := toStringSlice(row.Artists)

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

		// Use the PWO distance metric directly from PostgreSQL
		// The pwo_distance function provides position-weighted overlap scoring
		// Distance ranges from 0 (identical) to 1 (completely different)

		// Convert distance to similarity score (0-1 range, higher is better)
		distance := row.Distance
		score := float32(1.0 - distance)

		fmt.Printf("DEBUG: User %s - PWO distance: %.3f, similarity score: %.3f\n", row.Username, distance, score)

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
			Program:        "2Y",                                                                                     // Default program for the joke entry
			GraduationYear: 2026,                                                                                     // Default graduation year for the joke entry
			Overlap:        int32(0),                                                                                 // No overlap by design
			Score:          float32(0.0),                                                                             // No compatibility
			Artists:        []string{"Obscure artist 1", "Obscure artist 2", "Obscure artist 3", "Obscure artist 4"}, // Completely different taste
		}
		matches = append(matches, crushMatch)
	}

	return generated.Response(http.StatusOK, matches), nil
}

package business

import (
	"math"
	"sort"
	"strings"

	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// MatchingEngine handles music taste matching algorithms
type MatchingEngine struct {
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{}
}

// DedupeAndNormalize removes duplicates and empty strings from artist list
func (m *MatchingEngine) DedupeAndNormalize(in []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(in))
	for _, a := range in {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		al := strings.ToLower(a)
		if _, ok := seen[al]; ok {
			continue
		}
		seen[al] = struct{}{}
		out = append(out, a)
	}
	return out
}

// ComputeMatches finds users with similar music taste
func (m *MatchingEngine) ComputeMatches(target []string, caller string, users []*generated.User) []generated.MatchUser {
	// Build set for overlap scoring
	targetSet := make(map[string]struct{})
	for _, artist := range m.DedupeAndNormalize(target) {
		targetSet[strings.ToLower(artist)] = struct{}{}
	}

	targetSize := len(targetSet)
	if targetSize == 0 {
		return []generated.MatchUser{}
	}

	results := make([]generated.MatchUser, 0)

	for _, user := range users {
		// Skip the caller
		if user.Username == caller {
			continue
		}

		// Build user's artist set
		userSet := make(map[string]struct{})
		for _, artist := range m.DedupeAndNormalize(user.Artists) {
			userSet[strings.ToLower(artist)] = struct{}{}
		}

		userSize := len(userSet)
		if userSize == 0 {
			continue // Skip users with no artists
		}

		// Calculate overlap
		overlap := 0
		for artist := range targetSet {
			if _, found := userSet[artist]; found {
				overlap++
			}
		}

		if overlap == 0 {
			continue // Skip users with no overlap
		}

		// Calculate Jaccard similarity: |intersection| / |union|
		union := targetSize + userSize - overlap
		score := float64(overlap) / float64(union)

		// Round score to avoid floating point precision issues
		score = math.Round(score*10000) / 10000

		result := generated.MatchUser{
			Name:    user.FirstName + " " + user.LastName,
			Overlap: int32(overlap),
			Score:   float32(score),
		}

		results = append(results, result)
	}

	// Sort by score (descending), then by overlap (descending)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Overlap > results[j].Overlap
	})

	return results
}

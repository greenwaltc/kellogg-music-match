package business

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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
	cache        *similarityCache
}

// (range & limit now passed explicitly; context keys deprecated)

// similarityCache is a lightweight in-memory TTL cache for similarity results
type similarityCache struct {
	mu     sync.RWMutex
	ttl    time.Duration
	data   map[string]cacheEntry
	hits   uint64
	misses uint64
}
type cacheEntry struct {
	expires time.Time
	matches []*generated.MatchUser
}

func newSimilarityCache(ttl time.Duration) *similarityCache {
	return &similarityCache{ttl: ttl, data: make(map[string]cacheEntry)}
}
func (c *similarityCache) get(key string) ([]*generated.MatchUser, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expires) {
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}
	atomic.AddUint64(&c.hits, 1)
	return e.matches, true
}
func (c *similarityCache) set(key string, val []*generated.MatchUser) {
	c.mu.Lock()
	c.data[key] = cacheEntry{expires: time.Now().Add(c.ttl), matches: val}
	c.mu.Unlock()
}

// invalidateUser removes all cached similarity entries for a given user ID
func (c *similarityCache) invalidateUser(userID string) {
	prefix := "spotify:" + userID + ":"
	c.mu.Lock()
	for k := range c.data {
		if strings.HasPrefix(k, prefix) {
			delete(c.data, k)
		}
	}
	c.mu.Unlock()
}

// SimilarityCacheStats represents metrics about cache utilization
type SimilarityCacheStats struct {
	Hits    uint64
	Misses  uint64
	HitRate float64
}

func (c *similarityCache) stats() SimilarityCacheStats {
	h := atomic.LoadUint64(&c.hits)
	m := atomic.LoadUint64(&c.misses)
	total := h + m
	rate := 0.0
	if total > 0 {
		rate = float64(h) / float64(total)
	}
	return SimilarityCacheStats{Hits: h, Misses: m, HitRate: rate}
}

// NewMatchingService creates a new matching service with default config
func NewMatchingService(userRepo UserRepository, matching *MatchingEngine) *MatchingService {
	cfg := config.Load()
	ms := &MatchingService{
		userRepo:     userRepo,
		matching:     matching,
		artistConfig: &cfg.Artist,
		cache:        newSimilarityCache(30 * time.Second),
	}
	// Attempt to register invalidation hook if repository supports it
	if pr, ok := userRepo.(*PostgreSQLUserRepository); ok {
		pr.SetSpotifyArtistsUpdatedHook(ms.InvalidateSimilarityCache)
	}
	return ms
}

// NewMatchingServiceWithConfig creates a new matching service with provided config
func NewMatchingServiceWithConfig(userRepo UserRepository, matching *MatchingEngine, artistConfig *config.ArtistConfig) *MatchingService {
	if artistConfig == nil { // defensive default
		cfg := config.Load()
		artistConfig = &cfg.Artist
	}
	ms := &MatchingService{
		userRepo:     userRepo,
		matching:     matching,
		artistConfig: artistConfig,
		cache:        newSimilarityCache(30 * time.Second),
	}
	if pr, ok := userRepo.(*PostgreSQLUserRepository); ok {
		pr.SetSpotifyArtistsUpdatedHook(ms.InvalidateSimilarityCache)
	}
	return ms
}

// InvalidateSimilarityCache clears similarity entries for a user after new Spotify snapshot ingestion
func (s *MatchingService) InvalidateSimilarityCache(userID uuid.UUID) {
	if s.cache == nil {
		return
	}
	s.cache.invalidateUser(userID.String())
}

// SimilarityCacheStats exposes cache metrics (hits, misses, hit rate)
func (s *MatchingService) SimilarityCacheStats() *SimilarityCacheStats {
	if s.cache == nil {
		return &SimilarityCacheStats{}
	}
	st := s.cache.stats()
	return &st
}

// GetUserTopArtistsPage returns a page of the user's current Spotify top artists for a given range.
func (s *MatchingService) GetUserTopArtistsPage(ctx context.Context, userID uuid.UUID, rng string, limit, offset int32) (generated.ImplResponse, error) {
	cfg := config.Load()
	if rng == "" {
		rng = cfg.Matching.DefaultRange
	}
	allowed := false
	for _, ar := range cfg.Matching.AllowedRanges {
		if rng == ar {
			allowed = true
			break
		}
	}
	if !allowed {
		rng = cfg.Matching.DefaultRange
	}
	if limit <= 0 {
		limit = int32(cfg.Matching.DefaultLimit)
	}
	if max := cfg.Matching.MaxLimit; max > 0 && int(limit) > max {
		limit = int32(max)
	}
	items, err := s.userRepo.GetUserTopArtistsByRange(ctx, userID, rng, limit, offset)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: "failed to load top artists"}), nil
	}
	var totalCount int32
	if pr, ok := s.userRepo.(*PostgreSQLUserRepository); ok {
		if cnt, err := pr.CountUserTopArtistsByRange(ctx, userID, rng); err == nil {
			totalCount = cnt
		}
	}
	out := make([]generated.SpotifyTopArtistItem, 0, len(items))
	for _, it := range items {
		ai := generated.SpotifyTopArtistItem{
			SpotifyArtistId: it.SpotifyArtistID,
			Name:            it.Name,
			Rank:            it.Rank,
			Genres:          it.Genres,
		}
		if it.Popularity != nil {
			v := int32(*it.Popularity)
			// generated expects int32 already
			ai.Popularity = &v
		}
		if it.ImageURL != nil {
			ai.ImageUrl = it.ImageURL
		}
		out = append(out, ai)
	}
	hasMore := int32(len(items)) == limit
	if totalCount == 0 {
		// Fallback if count path not available (e.g., different repo implementation): best-effort estimate
		totalCount = offset + int32(len(items))
	}
	page := generated.TopArtistsPage{Items: out, HasMore: hasMore, TotalCount: totalCount}
	return generated.Response(http.StatusOK, page), nil
}

// GetUserTopTracksPage returns a page of the user's current Spotify top tracks for a given range.
func (s *MatchingService) GetUserTopTracksPage(ctx context.Context, userID uuid.UUID, rng string, limit, offset int32) (generated.ImplResponse, error) {
	cfg := config.Load()
	if rng == "" {
		rng = cfg.Matching.DefaultRange
	}
	allowed := false
	for _, ar := range cfg.Matching.AllowedRanges {
		if rng == ar {
			allowed = true
			break
		}
	}
	if !allowed {
		rng = cfg.Matching.DefaultRange
	}
	if limit <= 0 {
		limit = int32(cfg.Matching.DefaultLimit)
	}
	if max := cfg.Matching.MaxLimit; max > 0 && int(limit) > max {
		limit = int32(max)
	}
	items, err := s.userRepo.GetUserTopTracksByRange(ctx, userID, rng, limit, offset)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: "failed to load top tracks"}), nil
	}
	var totalCount int32
	if pr, ok := s.userRepo.(*PostgreSQLUserRepository); ok {
		if cnt, err := pr.CountUserTopTracksByRange(ctx, userID, rng); err == nil {
			totalCount = cnt
		}
	}
	out := make([]generated.SpotifyTopTrackItem, 0, len(items))
	for _, it := range items {
		ti := generated.SpotifyTopTrackItem{
			SpotifyTrackId: it.SpotifyTrackID,
			Name:           it.Name,
			Rank:           it.Rank,
			ArtistNames:    it.ArtistNames,
		}
		if it.Popularity != nil {
			v := int32(*it.Popularity)
			ti.Popularity = &v
		}
		if it.DurationMS != nil {
			v := int32(*it.DurationMS)
			ti.DurationMs = &v
		}
		if it.ImageURL != nil {
			ti.ImageUrl = it.ImageURL
		}
		if it.AlbumName != nil {
			ti.AlbumName = it.AlbumName
		}
		out = append(out, ti)
	}
	hasMore := int32(len(items)) == limit
	if totalCount == 0 {
		totalCount = offset + int32(len(items))
	}
	page := generated.TopTracksPage{Items: out, HasMore: hasMore, TotalCount: totalCount}
	return generated.Response(http.StatusOK, page), nil
}

// FindMusicMatches implements music matching business logic
// FindMusicMatches determines similar users based on Spotify top artists or tracks depending on the 'basis' value.
// BACKWARD COMPAT: existing generated wrapper currently does not pass a basis param; we infer from artistsRequest.Basis if later added or default to "artists".
func (s *MatchingService) FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, xUserUsername string, rng string, limit int32, overlapsLimit ...int32) (generated.ImplResponse, error) {
	if xUserUsername == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "username header is required"}), nil
	}

	// We ignore the manual artistsRequest now and rely on stored Spotify top artists.
	// Fetch the anchor user
	user, err := s.userRepo.GetUserByUsername(ctx, xUserUsername)
	if err != nil || user == nil {
		return generated.Response(http.StatusNotFound, generated.ErrorResponse{Message: "user not found"}), nil
	}

	// Apply defaults & validation
	cfg := config.Load() // lightweight call; could be injected if desired
	if rng == "" {
		rng = cfg.Matching.DefaultRange
	}
	allowed := false
	for _, ar := range cfg.Matching.AllowedRanges {
		if rng == ar {
			allowed = true
			break
		}
	}
	if !allowed {
		rng = cfg.Matching.DefaultRange
	}
	if limit <= 0 {
		limit = int32(cfg.Matching.DefaultLimit)
	}
	if max := cfg.Matching.MaxLimit; max > 0 && int(limit) > max {
		limit = int32(max)
	}

	// Determine basis (artists|tracks). Priority:
	// 1. business.MatchBasisContextKey{}
	// 2. legacy string key "match_basis" (deprecated)
	basis := "artists"
	if v := ctx.Value(MatchBasisContextKey{}); v != nil {
		if bs, ok := v.(string); ok && (bs == "artists" || bs == "tracks") {
			basis = bs
		}
	} else if v := ctx.Value("match_basis"); v != nil { // legacy fallback
		if bs, ok := v.(string); ok && (bs == "artists" || bs == "tracks") {
			basis = bs
		}
	}
	// Overridable via a sentinel field in ArtistsRequest future extension (defensive; ignore if empty)
	// NOTE: We avoid modifying generated model until OpenAPI regenerated; this allows early server adoption.

	// feature flag check for tracks
	if basis == "tracks" && !cfg.Matching.TracksEnabled {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "track-based matching disabled"}), nil
	}

	// Optional user name filter for fuzzy search of classmates/friends
	var nameFilter string
	if v := ctx.Value(MatchNameFilterContextKey{}); v != nil {
		if s, ok := v.(string); ok {
			nameFilter = strings.TrimSpace(s)
		}
	}

	// Determine overlaps limit (optional variadic for backward compatibility in tests not yet updated)
	var ovLimit int32 = 0
	if len(overlapsLimit) > 0 {
		ovLimit = overlapsLimit[0]
		if ovLimit < 0 { // negative treated as 0 (no limit)
			ovLimit = 0
		}
		if cfg.Matching.MaxOverlaps > 0 && int(ovLimit) > cfg.Matching.MaxOverlaps {
			ovLimit = int32(cfg.Matching.MaxOverlaps)
		}
	}

	cacheKey := fmt.Sprintf("spotify:%s:%s:%s:%d:%d:%s", basis, user.ID.String(), rng, limit, ovLimit, strings.ToLower(strings.TrimSpace(nameFilter)))
	// Cache lookup (guard against nil cache for safety if constructed elsewhere)
	if s.cache != nil {
		if matches, ok := s.cache.get(cacheKey); ok {
			return generated.Response(http.StatusOK, matches), nil
		}
	}
	// BEFORE performing similarity query, ensure the anchor user actually has Spotify top artist snapshots
	// for the requested range. If not, we short-circuit with an empty list (frontend shows a connect CTA)
	// instead of fabricating a pseudo self-match placeholder which could be misleading post-auth.
	var similar []SimilarUserResult
	if basis == "tracks" {
		if nameFilter != "" {
			similar, err = s.userRepo.FindSimilarUsersBySpotifyTopTracksFiltered(ctx, user.ID, rng, limit, nameFilter)
		} else {
			similar, err = s.userRepo.FindSimilarUsersBySpotifyTopTracks(ctx, user.ID, rng, limit)
		}
	} else {
		if nameFilter != "" {
			similar, err = s.userRepo.FindSimilarUsersBySpotifyTopArtistsFiltered(ctx, user.ID, rng, limit, nameFilter)
		} else {
			similar, err = s.userRepo.FindSimilarUsersBySpotifyTopArtists(ctx, user.ID, rng, limit)
		}
	}
	if err != nil {
		fmt.Printf("ERROR: similarity query failed: %v\n", err)
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: "similarity query failed"}), nil
	}

	matches := make([]*generated.MatchUser, 0, len(similar))
	for _, sim := range similar {
		fullName := strings.TrimSpace(sim.FirstName + " " + sim.LastName)
		// Build artist overlap names (no artificial cap; UI handles pagination/scroll)
		artistNames := make([]string, 0, len(sim.Overlaps))
		for _, ov := range sim.Overlaps {
			artistNames = append(artistNames, ov.Name)
		}
		// Overlap count = number of shared artists
		overlapCount := int32(len(sim.Overlaps))
		// Score normalization (revised): estimate upper bound of similarity for this particular overlap set.
		// Original version used sum(1/(2*anchor_rank)) which underestimated the true max and could inflate norm>1.
		// We now compute a per-overlap theoretical max using the best (smallest) rank between the two users; because
		// weight = 1/(anchor_rank + other_rank), the maximum possible weight for a given overlapping artist occurs when
		// the poorer (higher numeric) rank is improved to match the better (lower) rank. That idealized max weight becomes
		// 1/(best_rank + best_rank) = 1/(2*best_rank). Using best_rank ensures we never claim more than achievable if both
		// users had identical top positions for that artist. This tightens the bound while preventing inflation.
		var maxPossible float64
		for _, ov := range sim.Overlaps {
			if ov.AnchorRank <= 0 || ov.OtherRank <= 0 { // defensive guard
				continue
			}
			best := ov.AnchorRank
			if ov.OtherRank < best {
				best = ov.OtherRank
			}
			maxPossible += 1.0 / float64(2*best)
		}
		var score float32
		if overlapCount > 0 && maxPossible > 0 {
			norm := sim.Similarity / maxPossible
			if norm > 1 {
				// Debug log to help detect any remaining edge cases causing overflow beyond 1.
				fmt.Printf("DEBUG: normalized similarity >1 (%.4f) raw=%.6f maxPossible=%.6f overlaps=%d user=%s other=%s\n", norm, sim.Similarity, maxPossible, overlapCount, user.ID.String(), sim.UserID.String())
				if norm > 1.0000001 { // allow tiny FP epsilon
					norm = 1
				} else {
					// minor FP drift; clamp
					norm = 1
				}
			}
			score = float32(norm)
		}
		var gradYear int32 = 2025
		if sim.GraduationYear != nil {
			gradYear = *sim.GraduationYear
		}
		program := ""
		if sim.Program != nil {
			program = *sim.Program
		}
		// Guard against invalid data relative to OpenAPI constraints (Overlap must be >=1). Skip if none.
		if overlapCount == 0 {
			continue
		}
		mu := &generated.MatchUser{
			Name:           fullName,
			Program:        program,
			GraduationYear: gradYear,
			Overlap:        overlapCount,
			Score:          score,
			Artists:        artistNames,
			Overlaps: func() []generated.MatchUserOverlapsInner {
				// We stream through original sim.Overlaps preserving order (already sorted by rank sum in SQL)
				// Apply optional truncation if ovLimit > 0
				count := len(sim.Overlaps)
				if ovLimit > 0 && int(ovLimit) < count {
					count = int(ovLimit)
				}
				out := make([]generated.MatchUserOverlapsInner, 0, count)
				for i, ov := range sim.Overlaps {
					if i >= count {
						break
					}
					// Only include sane positive ranks
					if ov.AnchorRank <= 0 || ov.OtherRank <= 0 {
						continue
					}
					out = append(out, generated.MatchUserOverlapsInner{Name: ov.Name, AnchorRank: int32(ov.AnchorRank), OtherRank: int32(ov.OtherRank)})
				}
				return out
			}(),
		}
		// Include per-user top items depending on basis
		if basis == "artists" && len(sim.TopArtists) > 0 {
			arr := make([]generated.MatchUserTopArtistsInner, 0, len(sim.TopArtists))
			for _, a := range sim.TopArtists {
				// Guard rank
				if a.Rank <= 0 || a.Name == "" || a.SpotifyArtistID == "" {
					continue
				}
				arr = append(arr, generated.MatchUserTopArtistsInner{
					SpotifyArtistId: a.SpotifyArtistID,
					Name:            a.Name,
					Rank:            a.Rank,
				})
			}
			if len(arr) > 0 {
				mu.TopArtists = arr
			}
		}
		if basis == "tracks" && len(sim.TopTracks) > 0 {
			arr := make([]generated.MatchUserTopTracksInner, 0, len(sim.TopTracks))
			for _, t := range sim.TopTracks {
				if t.Rank <= 0 || t.Name == "" || t.SpotifyTrackID == "" {
					continue
				}
				arr = append(arr, generated.MatchUserTopTracksInner{
					SpotifyTrackId: t.SpotifyTrackID,
					Name:           t.Name,
					Rank:           t.Rank,
					ArtistNames:    t.ArtistNames,
				})
			}
			if len(arr) > 0 {
				mu.TopTracks = arr
			}
		}
		matches = append(matches, mu)
	}

	// New behavior: if no matches discovered, simply return an empty array (client displays hint/CTA)
	// This prevents a confusing "self" placeholder appearing before first successful sync.

	// Ensure deterministic order by the score we actually return to clients
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			if matches[i].Overlap == matches[j].Overlap {
				return matches[i].Name < matches[j].Name
			}
			return matches[i].Overlap > matches[j].Overlap
		}
		return matches[i].Score > matches[j].Score
	})

	if s.cache != nil {
		s.cache.set(cacheKey, matches)
	}
	return generated.Response(http.StatusOK, matches), nil
}

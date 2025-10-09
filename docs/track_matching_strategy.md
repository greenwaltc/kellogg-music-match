# Track-Based Matching Strategy

Date: 2025-10-08
Author: Strategy draft generated with assistant help
Status: Draft (ready for implementation planning)

## Objective
Extend the existing artist-based similarity system to also support track-based matching, allowing users to toggle between "Artists" and "Tracks" in the UI while reusing as much infrastructure (auth, caching, normalization) as practical.

## Summary Recommendation
Use a single polymorphic endpoint (`/findMusicMatches`) with an added required query parameter `type=artists|tracks` (alias `basis`). Keep internal implementations for artist and track similarity separate to allow independent evolution, but avoid proliferating public endpoints.

Later enhancements (hybrid / aggregated results) can build on this design without breaking existing clients.

## Alternatives Considered
| Option | Description | Pros | Cons | Decision |
|--------|-------------|------|------|----------|
| A. Single Endpoint (param `type`) | One route handles both bases | Minimal API sprawl, easy future extensibility, shared validation | Response schema must be flexible | **Chosen** |
| B. Two Endpoints | `/findMusicArtistMatches`, `/findMusicTrackMatches` | Tight semantic responses | Duplicated docs/tests, harder to extend | Rejected initially |
| C. Aggregated Endpoint | Return both artist + track matches in one call | One round trip for UI toggle | Heavier payload, slower first paint | Defer (possible layer on top) |

## Request & Response Contract (Proposed)
**Request**: `POST /findMusicMatches?type=tracks&range=medium_term&limit=50&overlapsLimit=10`
Body still accepts `{ "artists": [] }` (ignored for Spotify-based matching).

**Response**:
```json
{
  "basis": "tracks",
  "range": "medium_term",
  "matches": [
    {
      "name": "Jane Doe",
      "overlap": 7,
      "score": 0.62,
      "items": ["The Less I Know The Better", "Nikes"],
      "overlaps": [
        { "name": "The Less I Know The Better", "anchorRank": 5, "otherRank": 8 }
      ],
      "program": "MMM",
      "graduationYear": 2025
    }
  ],
  "meta": {
    "type": "tracks",
    "algorithmVersion": "tracks-1.0",
    "computedAt": "2025-10-08T13:44:02Z",
    "totalReturned": 25
  }
}
```
Backward compatibility: if `type` is omitted, default to `artists` (legacy behavior). UI migration can treat `artists` as default.

### Field Naming
- Use `items` (basis-agnostic) plus keep `artists` temporarily for compatibility (deprecated once UI switches).
- `basis` conveys the dimension used (`artists` or `tracks`).

## Similarity Algorithm Considerations
| Aspect | Artists | Tracks |
|--------|---------|--------|
| Snapshot Volatility | Moderate | Higher (rapid churn) |
| Overlap Frequency | Higher | Lower (sparser intersections) |
| Current Weight | `1/(r_a + r_b)` | Reuse, may add damping |
| Normalization | Sum weights / maxPossible | Same; maybe apply sparsity uplift |

### Track-Specific Adjustments (Phase 2+)
- Optional sparsity scaler: `adjusted = normalized * (1 + log1p(overlapCount)/K)`
- Potential recency weighting if track recency captured later.

## Data & SQL
Existing table: `spotify_top_track_snapshots` with ranks.
Add an overlap query analogous to artist version:
```sql
-- name: FindTopNSimilarUsersBySpotifyTracks :many
WITH anchor AS (
  SELECT v.user_id AS anchor_user_id, v.item_rank AS anchor_item_rank, v.spotify_track_id, v.name AS anchor_name
  FROM v_current_spotify_top_tracks v
  WHERE v.user_id = $1 AND v.range = $2
), others AS (
  SELECT v.user_id AS other_user_id, v.item_rank AS other_item_rank, v.spotify_track_id, v.name AS other_name
  FROM v_current_spotify_top_tracks v
  WHERE v.user_id <> $1 AND v.range = $2
), overlap AS (
  SELECT o.other_user_id,
         a.spotify_track_id,
         a.anchor_name AS name,
         a.anchor_item_rank AS anchor_rank,
         o.other_item_rank AS other_rank,
         1.0 / (a.anchor_item_rank + o.other_item_rank)::float8 AS weight
  FROM anchor a
  JOIN others o USING (spotify_track_id)
), agg AS (
  SELECT other_user_id,
         SUM(weight)::float8 AS similarity,
         jsonb_agg(jsonb_build_object(
           'spotify_track_id', spotify_track_id,
           'name', name,
           'anchor_rank', anchor_rank,
           'other_rank', other_rank
         ) ORDER BY (anchor_rank + other_rank), anchor_rank, other_rank) AS overlaps_json
  FROM overlap
  GROUP BY other_user_id
)
SELECT u.id AS user_id,
       u.username,
       u.first_name,
       u.last_name,
       u.program,
       u.graduation_year,
       agg.similarity,
       agg.overlaps_json
FROM agg
JOIN users u ON u.id = agg.other_user_id
WHERE agg.similarity > 0
ORDER BY agg.similarity DESC, u.created_at ASC
LIMIT $3;
```

### Index Review
Already present:
- `idx_spotify_top_track_user_range (user_id, range)`
Potential addition:
- `idx_spotify_top_track_snapshots_range_track (range, spotify_track_id)` for join selectivity (artist analog exists).

## Backend Architecture Changes
1. Extend repository interface:
```go
type UserRepository interface {
  FindSimilarUsersBySpotifyTopArtists(...)
  FindSimilarUsersBySpotifyTopTracks(...)
}
```
2. Add `FindSimilarUsersBySpotifyTopTracks` implementation via new sqlc query.
3. Update `MatchingService.FindMusicMatches`:
   - Parse `type` param (default `artists`).
   - Select appropriate repo method.
   - Add `basis` to response and to cache key.
4. Cache key pattern: `spotify:{userID}:{basis}:{range}:{limit}:{ovLimit}`.
5. Shared match construction helper returns list; normalization logic reused.
6. Add `basis` to OpenAPI spec (enum `artists|tracks`).

## Caching & Performance
| Concern | Approach |
|---------|----------|
| Higher track churn | Same 30s TTL initially; monitor hit rate per basis |
| Cache explosion | Basis included in key; optional LRU/eject older basis entries per user |
| First toggle latency | Lazy fetch; optionally prefetch second basis after first response (config flag) |

## UI & UX
### Toggle Design
- Segmented control (Artists | Tracks) above range controls.
- Persist selection in `localStorage (kmmMatchBasis)`.
- Show skeleton only on first fetch for a basis+range.

### Label Changes
- Heading changes to "Your Top Music Matches (Artists)" / "(Tracks)" or keep a single heading and update the artists/tracks section label.
- Section title: "Artists in Common" → "Tracks in Common".
- Rank tuple tooltip unchanged semantics, just refers to the selected basis ranks.

### Empty / Sparse States
- Tracks often sparser: add hint when overlap < 2: "Few shared tracks—try Artists for a broader taste profile." (Follow pattern used for low short-term artist overlap.)

### Accessibility
- Toggle buttons: `role="tab"`, `aria-selected`.
- ARIA live region for basis switch announcing new basis loaded.

## Rollout Plan
1. Backend query + repo method for tracks (behind feature flag `MATCH_TRACKS_ENABLED`).
2. Add `type` param to existing endpoint; ignore if flag disabled (return 400 for `tracks` until enabled).
3. UI toggle (tracks option disabled / tooltip “Coming soon” if flag false, flag exposed via config endpoint).
4. Metrics logging per basis: similarity compute ms, average overlap size, cache hit rate.
5. After data stable: remove flag & update docs.
6. Later: optional aggregated endpoint or prefetch optimization.

## Testing Strategy
### Backend
- Unit tests replicate artist tests for tracks (ordering, normalization ≤ 1, overlap truncation logic if applied).
- Integration test with synthetic overlapping track snapshots across ranges.
- Negative tests (no overlap returns empty list, not placeholder).

### Frontend
- Service tests: fetch with `type=artists` then `type=tracks`; verify separate caching.
- Component tests: toggle basis updates label, does not refetch if cached, does fetch if missing.
- Hint tests for sparse track overlap.
- E2E: Toggle stress (rapid switching) yields consistent item render.

## Potential Future Enhancements
| Idea | Description | Rationale |
|------|-------------|-----------|
| Hybrid Score | Combine artist + track similarity (weighted) | Richer signal for compatibility |
| Genre Layer | Aggregate overlapping genres from track artists | Explains why matches appear |
| Preview Clips | Include 30s preview URLs for track overlaps | Engagement & delight |
| Delta Badges | Show +N additional overlaps when switching basis | Encourages exploration |

## Risks & Mitigations
| Risk | Mitigation |
|------|-----------|
| Track overlap too sparse → low perceived value | Provide contextual hint; consider hybrid default if engagement low |
| Performance regression from second query path | Add timing logs & compare p95 vs artist path before enabling prefetch |
| Complexity creep in single endpoint | Isolate per-basis logic in small internal functions; keep switch minimal |
| Cache fragmentation | Track basis adoption metrics; optionally increase TTL for basis with fewer hits |

## Implementation Checklist (Engineering)
**Backend**
- [ ] SQL query file & migration for track similarity (+ index if needed)
- [ ] sqlc generation
- [ ] Repo method + tests
- [ ] MatchingService switch w/ cache key expansion
- [ ] OpenAPI spec update (query param + response field `basis` / `items`)
- [ ] Feature flag gating
- [ ] Logging & metrics

**Frontend**
- [ ] Add basis signal & persistence
- [ ] Extend `MatchService.fetch` to include `type`
- [ ] Toggle UI + accessible roles
- [ ] Update templates (label swap, items binding instead of artists where basis=tracks)
- [ ] Track sparse hint
- [ ] Tests (unit + integration) updated

**Docs & Ops**
- [ ] README / API docs updated
- [ ] Rollout comms & feature flag instructions
- [ ] Monitoring dashboard panels (hit rate, overlap distribution) per basis

## Decision Log
| Date | Decision | Notes |
|------|----------|-------|
| 2025-10-08 | Adopt single endpoint with `type` param | Leaves door open for hybrid & aggregation |

---
This document should be updated as real performance data and user feedback arrive. Keep changes additive; note revisions in the Decision Log.

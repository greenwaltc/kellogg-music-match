# Requirements Coverage Snapshot (Spotify Matching & Related Features)

_Last updated: 2025-10-07_

## 1. Implemented / Satisfied

- Spotify OAuth with PKCE (frontend + backend) and token persistence (encrypted when key present).
- Spotify top artists & tracks ingestion for ranges: short_term, medium_term, long_term.
- Rank‑weighted similarity (Σ 1/(anchor_rank + other_rank)) with corrected normalization & clamping.
- OpenAPI: range, limit, overlapsLimit parameters; structured overlaps (name, anchorRank, otherRank).
- Similarity cache (TTL) + invalidation on new artist snapshots.
- SQL performance indices (range+artist, rank).
- Repository adapter decoding JSON overlaps; unified business layer call.
- Score normalization fix using per-overlap clamped upper bound.
- Backend refactor: explicit params (removed context key hacks), config-driven defaults & caps.
- Frontend Matches page:
  - Range toggle (short/medium/long) with tooltips & persisted state (localStorage + query params).
  - Adjustable limit, overlapsLimit, debounce; reset control.
  - Pagination and Show All toggle for overlaps.
  - Loading shimmer, skeleton states, reduced-motion compliance.
  - Rank tuple display (#anchor/#other).
  - Deep-linking (URL sync) + local preference persistence.
  - SCSS modularization (partial + namespace) and tooltip directive reuse.
  - Structured overlap consumption (ranks displayed).
- Tests:
  - Similarity ordering & normalization (non-inflation).
  - Overlaps truncation & ordering.
  - Password reset suite restored.
  - Consolidated single Ginkgo suite (no duplicate RunSpecs).
- Logging:
  - Detailed token exchange errors.
  - Normalization debug when raw > 1.
  - Spotify sync 403 visibility.
- Post-auth automatic matches refresh; removed stale /api path mismatch.
- Server-side cap for overlapsLimit + param exposed.
- Debounced range/limit changes; inflight request guard.

## 2. Partially Done / Needs Enhancement

| Area | Current State | Improvement Needed |
|------|---------------|--------------------|
| Track-based similarity usage | Tracks ingested, not used in scoring | Add mode=tracks / combined weighting |
| Normalization documentation | Tooltip only | Add README / OpenAPI description |
| Metrics / Observability | Cache stats in memory | Expose /metrics or /admin JSON |
| Token lifecycle | Persisted, no proactive refresh | Add scheduled refresh & failure alerting |
| Error UX | Logs only for 403 per-range failures | Surface partial failure status to UI |
| Snapshot retention | No pruning | Implement retention policy (time or count) |
| Accessibility | Basic tooltips | aria-live for updates, focus management |
| Tests breadth | Core similarity covered | End-to-end multi-range ingest + match |
| Tooltip directive | No unit tests | Add directive behavior tests |
| Config loading | Per-call load | Inject once at service construction |
| API docs | OverlapsLimit added, limited examples | Provide concrete examples & schema samples |
| Cache metrics | Internal only | Export Prometheus format or structured JSON |
| Security tests | Assumed encryption path | Test token encryption/rotation explicitly |

## 3. Not Yet Started / Future Features

- Track-only and combined (artists+tracks) similarity modes.
- Weighted multi-range blending (e.g., 0.5 medium + 0.3 long + 0.2 short).
- Match explanation endpoint (per-artist contribution weights).
- User privacy controls (opt-out from being matched).
- Admin tools: re-run ingestion, revoke tokens, inspect errors.
- Rate limiting / adaptive backoff & retry classification.
- Snapshot pruning / archival strategy.
- Match list pagination (for large user base).
- API versioning / backward compatibility policy.
- Feature flags for algorithm variants.
- SLO metrics (latency, cache hit rate exports).
- Light theme or theming system (SCSS tokens / CSS vars).
- Circuit breaker for repeated 403 during sync.
- Display last sync timestamp & range statuses in UI.
- Prometheus/OpenTelemetry integration.
- Artist ID deep-linking (future detail pages).
- Shareable match links / social preview metadata.

## 4. High-Impact Next Steps (Suggested Sequence)

1. Tracks & Combined Mode  
   - Add mode param (?mode=artists|tracks|combined).  
   - Compute separate similarity vectors and merge for combined.
2. Metrics & Observability  
   - /internal/metrics: cache hits/misses, avg similarity query ms, ingestion successes/failures.  
   - Optional Prometheus scrape endpoint.
3. Snapshot Retention  
   - Policy: keep last N (e.g., 5) snapshots per (user, range).  
   - Remove older rows via scheduled job or on insert.
4. Explanations API  
   - Return per-overlap contribution: weight = 1/(a_rank+o_rank), normalized fraction of total.
5. Proactive Refresh Flow  
   - Background job refreshes tokens & schedules periodic top-items refresh (e.g., daily per range).

## 5. Current Configurable Limits

| Setting | Source | Default |
|---------|--------|---------|
| Default match range | MATCHING_DEFAULT_RANGE | medium_term |
| Default limit | MATCHING_DEFAULT_LIMIT | 10 (UI overrides to 50) |
| Max limit | MATCHING_MAX_LIMIT | 50 |
| Max overlaps | MATCHING_MAX_OVERLAPS | 100 |
| OverlapsLimit (client) | Query param | User-selected (capped) |

## 6. Risk & Technical Debt Snapshot

| Risk | Impact | Mitigation |
|------|--------|------------|
| Missing token refresh | Expired tokens reduce ingestion | Add refresh scheduler |
| No pruning | Table growth & slower joins | Implement retention + index maintenance |
| Limited metrics | Blind spots in performance | Add metrics endpoint |
| Single similarity mode | User feature gap | Add tracks/combined toggle |
| No explicit error UI | User confusion on partial failures | Expose statuses & messages |
| Unbounded overlap payload | Large payload sizes | Enforce server & client overlap caps |

## 7. Quick Win Checklist

- [ ] Add /internal/metrics (JSON) + cache stats.
- [ ] Add mode param (artists default).
- [ ] Prune snapshots older than X (migration + job).
- [ ] Expose last sync timestamp in match response.
- [ ] Document normalization formula in README.
- [ ] Unit test tooltip directive (hover + keyboard).
- [ ] Add retain/recompute endpoint for manual admin trigger.

## 8. Normalization Formula (Documented)

Let:
- Overlaps = set of shared artists.
- For each overlap i: weight_i = 1 / (anchor_rank_i + other_rank_i)
- Max_i = 1 / (2 * best_rank_i) where best_rank_i = min(anchor_rank_i, other_rank_i)
Similarity_raw = Σ weight_i  
Similarity_norm = min(1, Similarity_raw / Σ Max_i)

## 9. Data Model Notes

Tables:
- spotify_top_artist_snapshots(user_id, range, spotify_artist_id, item_rank, taken_at, ...)
- spotify_top_track_snapshots(user_id, range, spotify_track_id, item_rank, taken_at, ...)
Indices support querying by (user_id, range) and overlap joins by artist_id + range.

## 10. Open Questions (For Future Design)

- Should rank decay weighting be nonlinear (e.g., 1/log(rank+1))?
- Should we support partial weighting across ranges (user configurable)?
- How to prevent “self-reinforcement” if future features alter ranking data?
- Do we need privacy tiers (hide overlaps but show score)?

---
### To Update This File
Regenerate after major changes:
- Add or remove features
- Modify normalization
- Introduce new API params

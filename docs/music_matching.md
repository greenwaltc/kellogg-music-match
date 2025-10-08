# Music Matching Algorithm

This document describes the current production music similarity scoring system and contrasts it with the legacy Position‑Weighted Overlap (PWO) PostgreSQL function retained for historical benchmarking.

## Overview
The live system computes similarity between a requesting ("anchor") user and candidate users using their stored Spotify Top Artists lists for a selected time range (`short_term`, `medium_term`, or `long_term`).

Key characteristics:
- Rank-aware: Higher (closer to 1) artist ranks contribute more.
- Symmetry-respecting: Order difference influences weight but cannot exceed 1 after normalization.
- Stable bounds: Normalization guarantees scores in [0,1].
- Configurable: Server-side defaults and maximum caps for `limit` and `overlapsLimit`.
- Performant: In-memory 30s TTL cache keyed by `(user, range, limit, overlapsLimit)` invalidated when new Spotify snapshot ingests.

## Data Inputs
For each user and time range we maintain an ordered list of top artists with 1-based ranks. Overlaps are computed in SQL returning triplets: `(artistName, anchorRank, otherRank)` sorted by `(anchorRank + otherRank)` ascending (best combined prominence first).

## Scoring Steps
Given the overlap set O for a pair (A,B):

1. Raw Per-Artist Weight:  w_i = 1 / (anchorRank_i + otherRank_i)
2. Theoretical Max For That Artist: max_i = 1 / (2 * min(anchorRank_i, otherRank_i))  (the case where the worse rank improved to the better one)
3. Aggregate:
   rawSimilarity = Σ w_i
   maxPossible   = Σ max_i
4. Normalized Score: score = clamp(rawSimilarity / maxPossible, 0, 1)

If there are zero overlaps the pair is skipped (not returned). Floating point drift above 1 is clamped with a small epsilon.

### Rationale
Using `min(anchorRank, otherRank)` tightens the theoretical ceiling compared to earlier simplifications (e.g., using only anchor rank) and prevents inflation where asymmetric rank pairs produced normalized values slightly >1.

### Structured Overlaps
Each match includes an `overlaps` array with:
```json
{ "name": "Artist Name", "anchorRank": 2, "otherRank": 5 }
```
The list is optionally truncated by `overlapsLimit` (after ordering) to minimize payload size and UI clutter while preserving rank metadata.

## Query Parameters
- range: Spotify time range (`short_term | medium_term | long_term`), default from configuration.
- limit: Max number of users to return (bounded by `Matching.MaxLimit`).
- overlapsLimit: Optional truncation of returned overlap list per user (bounded by `Matching.MaxOverlaps`).

The request body for `/findMusicMatches` is currently ignored (legacy format kept for backward compatibility) because Spotify-derived lists are authoritative.

## Caching
A lightweight in-memory cache stores full match responses for 30 seconds keyed by `spotify:{userID}:{range}:{limit}:{overlapsLimit}`. Invalidation occurs automatically when the repository ingests a new snapshot of the user’s Spotify top artists.

## Legacy PWO Function (Deprecated for Live Scoring)
The migration `V010__pwo_metric.sql` defines `pwo_distance(artist_array_a, artist_array_b, alpha)` returning a distance in [0,1]. Historical approach:
- Similarity previously defined as `1 - pwo_distance`.
- Alpha parameter adjusted position sensitivity.
- Operated entirely within PostgreSQL.

Reasons for Transition:
- Needed structured overlap metadata with both ranks for richer UI.
- Desire for per-overlap theoretical max normalization for transparent reasoning.
- Easier evolution of algorithm in Go without changing DB function signature.

The legacy function remains for experimentation or potential hybrid scoring (future A/B testing).

## Testing
Behavioral tests cover:
- Normalization never exceeds 1.
- Overlap truncation ordering fidelity.
- Time range isolation (no mixing of short/medium/long term lists).
- Identical preference edge cases (score = 1 and full overlap ordering).
- Cache key differentiation when overlapsLimit changes.

## Future Directions
Planned enhancements include:
- Optional dual reporting: { currentScore, legacyPwoScore } for analytics.
- Popularity-aware weighting (down-weight ubiquitous artists).
- Temporal decay or recency boost for short_term data.
- Confidence metrics when overlap count is low.

## Summary
Current scoring = rank-weighted reciprocal sum normalized against a realistic per-overlap ceiling derived from best achievable ranks. This yields intuitive, stable similarity values, preserves structured overlap detail, and supports performant caching and configurability.

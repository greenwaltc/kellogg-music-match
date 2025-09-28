# MusicBrainz CSV Deduplication

## Overview

The original `musicbrainz_artists_50k.csv` file contained **2,548 exact duplicate rows** (same MusicBrainz ID and all other fields). This caused issues with the database loading process when combined with unique constraints.

## Deduplication Process

Used `scripts/deduplicate_csv.py` to remove exact duplicates while preserving legitimate artists who happen to share the same name but have different MusicBrainz IDs.

## Results

| File | Rows | Description |
|------|------|-------------|
| `musicbrainz_artists_50k.csv` | 50,001 | Original file with duplicates (including header) |
| `musicbrainz_artists_50k_deduplicated.csv` | 47,453 | Deduplicated file (including header) |

- **Removed**: 2,548 exact duplicate rows
- **Kept**: 47,452 unique artist records
- **Remaining name duplicates**: 251 (legitimate different artists with same names)

## Files Updated

- `docker-compose.yml`: Updated to use `musicbrainz_artists_50k_deduplicated.csv`
- `scripts/deduplicate_csv.py`: Created deduplication utility
- `backend/db/schema/migrations/V019__fix_musicbrainz_upsert_function.sql`: Fixed upsert function to remove non-existent `updated_at` column references

## Example of Legitimate Name Duplicates (Kept)

```csv
9adda98d-1be8-4865-86e4-03aceeb88bed,忍,Shinobu,Person,male,,,,composer & arranger,54
ea57492c-d40f-4d3e-854b-32bddfe40fe9,忍,Shinobu,Person,,,,,Vaporwave producer on bandcamp,49
```

These have the same name but different MusicBrainz IDs and descriptions, so they are legitimately different artists.

## Database Schema Fix

The original `upsert_artist_with_musicbrainz` function in migration V011 contained references to a non-existent `updated_at` column, causing the upsert operations to fail when trying to update existing artists.

**Fixed in V019**: Created a new migration that properly replaces the function without the invalid column references, following Flyway best practices of not modifying existing migrations.

## Benefits

1. **Faster Loading**: Reduces processing time by ~5%
2. **Cleaner Data**: Eliminates redundant entries
3. **Better Performance**: Reduces unnecessary database operations
4. **Consistent Results**: Ensures predictable loading behavior
5. **Zero Errors**: Fixed upsert function eliminates database constraint violations
# MusicBrainz Artist Fetcher

This script fetches artist data from the MusicBrainz API with pagination and saves it to JSON or CSV format.

## Files Created

- `fetch_musicbrainz_artists.py` - Main script for fetching artists
- `requirements.txt` - Python dependencies
- `musicbrainz_artists_1k.json` - Sample dataset with 1000 artists (JSON format)
- `musicbrainz_artists_1k.csv` - Sample dataset with 1000 artists (CSV format)

## Usage

### Basic Usage

```bash
# Fetch 1000 artists and save as JSON
python3 scripts/fetch_musicbrainz_artists.py --max-artists 1000 --format json

# Fetch 5000 artists and save as CSV
python3 scripts/fetch_musicbrainz_artists.py --max-artists 5000 --format csv

# Fetch all artists (WARNING: This could take hours and result in millions of records)
python3 scripts/fetch_musicbrainz_artists.py --format json --delay 1.0
```

### Command Line Options

- `--max-artists N` - Maximum number of artists to fetch (default: unlimited)
- `--delay N` - Delay between API requests in seconds (default: 1.0)
- `--format json|csv` - Output format (default: json)
- `--output FILENAME` - Custom output filename (auto-generated if not provided)

### Examples

```bash
# Quick test with 10 artists
python3 scripts/fetch_musicbrainz_artists.py --max-artists 10 --output test.json

# Large dataset with faster requests (be respectful!)
python3 scripts/fetch_musicbrainz_artists.py --max-artists 10000 --delay 0.5 --format csv

# Custom filename
python3 scripts/fetch_musicbrainz_artists.py --max-artists 5000 --output my_artists.json
```

## Data Fields

Each artist record contains:

- `id` - MusicBrainz unique identifier
- `name` - Artist name
- `sort_name` - Name formatted for sorting
- `type` - Artist type (Person, Group, Other, etc.)
- `gender` - Gender (for persons)
- `country` - Country code
- `life_span_begin` - Birth/formation date
- `life_span_end` - Death/dissolution date
- `disambiguation` - Additional info to distinguish similar names
- `score` - Relevance score from MusicBrainz

## Rate Limiting

The script includes built-in rate limiting to be respectful to the MusicBrainz API:

- Default 1.0 second delay between requests
- Configurable via `--delay` parameter
- User-Agent string identifies the application

## Performance Estimates

- ~100 artists per request
- With 1.0s delay: ~360 artists per hour
- With 0.5s delay: ~720 artists per hour
- 10,000 artists: ~14-28 hours depending on delay
- 100,000 artists: ~5-12 days depending on delay

## Sample Data

The included sample files (`musicbrainz_artists_1k.*`) contain 1000 artists including:
- Various Artists
- Classical composers (Bach, Mozart, etc.)
- Popular artists (Bruce Springsteen, etc.)
- International artists
- Different artist types (Person, Group, Other)

## MusicBrainz API

- Documentation: https://musicbrainz.org/doc/MusicBrainz_API
- Rate limiting: Be respectful, use delays
- No API key required for basic queries
- Data is under Creative Commons license

## Integration with Kellogg Music Match

This artist data can be used to:

1. **Populate search suggestions** in the UI
2. **Validate artist names** during user input
3. **Enrich user profiles** with additional artist metadata
4. **Improve matching algorithms** with standardized artist data
5. **Provide artist disambiguation** for common names

### Loading into Database

```sql
-- Example: Create a reference artists table
CREATE TABLE artist_reference (
    musicbrainz_id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sort_name VARCHAR(255),
    artist_type VARCHAR(50),
    country CHAR(2),
    disambiguation TEXT
);

-- Import from CSV (PostgreSQL example)
COPY artist_reference FROM '/path/to/musicbrainz_artists.csv' 
WITH (FORMAT csv, HEADER true);
```
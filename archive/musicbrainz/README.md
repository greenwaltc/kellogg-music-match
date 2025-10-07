Archived MusicBrainz Integration Assets

These files were part of the legacy MusicBrainz reference artist ingestion pipeline. The application has migrated to a Spotify-derived artist preference model.

Contents:
- fetch_musicbrainz_artists.py
- load_musicbrainz_artists.py
- load_musicbrainz_data.sh
- load_musicbrainz_k8s.sh
- load_artists_k8s.sh
- deduplicate_csv.py
- README_musicbrainz.md
- README_integration.md
- requirements.txt (Python deps for the above)
- Sample datasets (if retained): musicbrainz_artists_50k_deduplicated.csv (removed unless explicitly restored)

Retention Rationale:
Keeping these for historical reference and potential data backfill experiments. They are no longer invoked by builds, Pulumi, or docker-compose.

Deletion Safety:
All runtime references (Pulumi, backend code, docker-compose) have been removed. Safe to delete entirely if repository size or clarity becomes a concern.

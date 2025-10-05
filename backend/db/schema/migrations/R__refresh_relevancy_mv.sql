-- R__refresh_relevancy_mv.sql (Repeatable Migration)
-- Purpose: Provide a lightweight, idempotent periodic refresh hook for the
--          concert_event_relevancy_mv materialized view plus ancillary indexes.
-- Execution: Flyway will re-run this file whenever its checksum changes.
-- Strategy: Refresh CONCURRENTLY to avoid read blocking; if the MV does not
--           yet exist (earlier baseline), the REFRESH will fail, so guard.
-- Note: Adjust schedule externally (CI/CD or cron) by touching this file to
--       trigger re-execution if you rely solely on Flyway runs.

DO $$
BEGIN
  -- Ensure MV exists before attempting refresh
  IF EXISTS (
    SELECT 1 FROM pg_matviews WHERE matviewname = 'concert_event_relevancy_mv'
  ) THEN
    BEGIN
      EXECUTE 'REFRESH MATERIALIZED VIEW CONCURRENTLY concert_event_relevancy_mv';
    EXCEPTION WHEN feature_not_supported THEN
      -- Fallback if not created with unique index or during initial load
      EXECUTE 'REFRESH MATERIALIZED VIEW concert_event_relevancy_mv';
    END;
  END IF;
END;$$;

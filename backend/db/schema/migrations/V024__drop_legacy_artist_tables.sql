-- V024: Drop legacy artist matching tables now superseded by Spotify integration
-- This migration irreversibly removes manual artist entry and similarity artifacts.
-- Order matters due to foreign keys / dependencies.

-- Safety: Drop triggers/functions that might reference these tables first (if they still exist)
DO $$ BEGIN
  EXECUTE 'DROP FUNCTION IF EXISTS artist_neighbors_mark_stale() CASCADE';
EXCEPTION WHEN undefined_function THEN END; $$;

-- Helper DO block to drop an object that may have been created as table, materialized view, or view
DO $$
DECLARE
  obj text;
  objs text[] := ARRAY['artist_listener_counts','artist_neighbors','user_submitted_artists','user_artists','reference_artists','artists'];
  is_table bool;
  is_matview bool;
  is_view bool;
BEGIN
  FOREACH obj IN ARRAY objs LOOP
    SELECT EXISTS (
             SELECT 1 FROM pg_catalog.pg_class c
             JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
             WHERE c.relkind = 'r' AND c.relname = obj AND n.nspname = 'public'
           ) INTO is_table;
    SELECT EXISTS (
             SELECT 1 FROM pg_catalog.pg_matviews m
             WHERE m.schemaname = 'public' AND m.matviewname = obj
           ) INTO is_matview;
    SELECT EXISTS (
             SELECT 1 FROM pg_catalog.pg_views v
             WHERE v.schemaname = 'public' AND v.viewname = obj
           ) INTO is_view;

    IF is_table THEN
      EXECUTE format('DROP TABLE IF EXISTS %I CASCADE', obj);
    ELSIF is_matview THEN
      EXECUTE format('DROP MATERIALIZED VIEW IF EXISTS %I CASCADE', obj);
    ELSIF is_view THEN
      EXECUTE format('DROP VIEW IF EXISTS %I CASCADE', obj);
    END IF;
  END LOOP;
END $$;

-- Some systems may have created a sequence for artists; drop if present
DO $$ BEGIN
  EXECUTE 'DROP SEQUENCE IF EXISTS artists_id_seq CASCADE';
EXCEPTION WHEN undefined_table THEN END; $$;

-- Document rationale
COMMENT ON SCHEMA public IS 'Legacy artist preference tables removed in favor of Spotify-based automatic preference inference.';

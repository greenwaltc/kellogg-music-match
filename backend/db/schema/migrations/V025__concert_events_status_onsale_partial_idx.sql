-- V025__concert_events_status_onsale_partial_idx.sql
-- Purpose: Add a partial index to accelerate frequent predicates
--          ce.status = 'onsale' present in event listing queries.
-- Rationale: Most read queries filter WHERE ce.status = 'onsale'. If the
--            table grows and includes historical/off-sale rows, this keeps
--            visibility map and heap scans small.
-- Safety: IF NOT EXISTS for idempotence. Index is small because it only
--         stores PKs for qualifying rows.
-- Rollback: DROP INDEX IF EXISTS idx_concert_events_status_onsale;

CREATE INDEX IF NOT EXISTS idx_concert_events_status_onsale
  ON concert_events (id)
  WHERE status = 'onsale';

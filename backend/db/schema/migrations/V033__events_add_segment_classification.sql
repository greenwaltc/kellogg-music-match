-- V033: Add event segment and classification fields for on-demand events
-- Stores high-level type (segment) and finer classification names as provided by Ticketmaster

ALTER TABLE events
    ADD COLUMN IF NOT EXISTS segment_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS classification_name VARCHAR(100);

-- Optional indexes to support filtering by segment/classification
CREATE INDEX IF NOT EXISTS idx_events_segment_name ON events(segment_name);
CREATE INDEX IF NOT EXISTS idx_events_classification_name ON events(classification_name);

COMMENT ON COLUMN events.segment_name IS 'High-level Ticketmaster segment (e.g., Music, Sports, Arts & Theatre)';
COMMENT ON COLUMN events.classification_name IS 'Finer classification/type within the segment (e.g., Rock, Football)';

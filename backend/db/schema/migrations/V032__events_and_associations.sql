-- V032: On-demand events and user associations
-- Introduces minimal snapshot tables for on-demand Ticketmaster events
-- and user associations overlay. These coexist with legacy concert_* tables
-- and do not affect existing Chicago ingest flows.

-- Events table: minimal normalized snapshot for identified events
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source VARCHAR(50) NOT NULL DEFAULT 'ticketmaster', -- event source system
    external_id VARCHAR(255) NOT NULL,                 -- ID from external source
    name VARCHAR(1000) NOT NULL,
    venue VARCHAR(500),
    city VARCHAR(100),
    state VARCHAR(50),
    country VARCHAR(50),
    start_utc TIMESTAMPTZ NOT NULL,
    url VARCHAR(2000),
    raw_json JSONB,                                    -- minimal snapshot payload
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT events_source_chk CHECK (source IN ('ticketmaster'))
);

-- Uniqueness across source+external id to support multiple providers in future
CREATE UNIQUE INDEX IF NOT EXISTS ux_events_source_external ON events(source, external_id);
-- Index to support chronological queries
CREATE INDEX IF NOT EXISTS idx_events_start_utc ON events(start_utc);

-- Maintain updated_at timestamp automatically (function defined in earlier migrations)
CREATE TRIGGER trg_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- User associations to events (INTERESTED | GOING | LOOKING_FOR_GROUP)
CREATE TABLE IF NOT EXISTS user_event_associations (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, event_id),
    CONSTRAINT valid_user_event_status CHECK (
        status IN ('INTERESTED', 'GOING', 'LOOKING_FOR_GROUP')
    )
);

-- updated_at trigger
CREATE OR REPLACE FUNCTION update_user_event_associations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_user_event_associations_updated_at
    BEFORE UPDATE ON user_event_associations
    FOR EACH ROW EXECUTE FUNCTION update_user_event_associations_updated_at();

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_user_event_assoc_event ON user_event_associations(event_id);
CREATE INDEX IF NOT EXISTS idx_user_event_assoc_user ON user_event_associations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_event_assoc_event_status ON user_event_associations(event_id, status);

-- Documentation
COMMENT ON TABLE events IS 'Minimal on-demand event snapshot (Ticketmaster and future sources). Rows exist only when at least one user association is present.';
COMMENT ON COLUMN events.raw_json IS 'Minimal raw event payload snapshot for rendering; not a full denormalization.';
COMMENT ON TABLE user_event_associations IS 'User association state for events: INTERESTED | GOING | LOOKING_FOR_GROUP';

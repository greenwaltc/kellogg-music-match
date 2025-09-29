-- V020: Create concert events, venues, and artists tables for Ticketmaster API integration
-- This migration creates the database schema to store concert events fetched from external APIs

-- Create venues table
CREATE TABLE IF NOT EXISTS venues (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    street VARCHAR(500),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(50),
    country VARCHAR(50) NOT NULL,
    postal VARCHAR(20),
    capacity INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create concert_artists table (separate from musicbrainz artists)
CREATE TABLE IF NOT EXISTS concert_artists (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    genres TEXT[], -- Array of genre strings
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create concert_events table
CREATE TABLE IF NOT EXISTS concert_events (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(1000) NOT NULL,
    event_date TIMESTAMP NOT NULL,
    venue_id VARCHAR(255) REFERENCES venues(id) ON DELETE SET NULL,
    genres TEXT[], -- Array of genre strings
    price_min DECIMAL(10,2),
    price_max DECIMAL(10,2),
    price_currency VARCHAR(10) DEFAULT 'USD',
    ticket_url VARCHAR(2000),
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'unknown', -- onsale, offsale, cancelled, etc.
    age_restriction VARCHAR(100),
    provider VARCHAR(50) NOT NULL DEFAULT 'ticketmaster', -- API provider source
    external_url VARCHAR(2000), -- Original event URL from provider
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_status CHECK (status IN ('onsale', 'offsale', 'cancelled', 'postponed', 'rescheduled', 'unknown')),
    CONSTRAINT valid_price_range CHECK (price_min IS NULL OR price_max IS NULL OR price_min <= price_max),
    CONSTRAINT future_event_date CHECK (event_date > CURRENT_TIMESTAMP - INTERVAL '1 day') -- Allow events from yesterday
);

-- Create junction table for event-artist relationships (many-to-many)
CREATE TABLE IF NOT EXISTS concert_event_artists (
    event_id VARCHAR(255) REFERENCES concert_events(id) ON DELETE CASCADE,
    artist_id VARCHAR(255) REFERENCES concert_artists(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'performer', -- performer, headliner, opening_act, etc.
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (event_id, artist_id)
);

-- Create indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_concert_events_date ON concert_events(event_date);
CREATE INDEX IF NOT EXISTS idx_concert_events_venue ON concert_events(venue_id);
CREATE INDEX IF NOT EXISTS idx_concert_events_status ON concert_events(status);
CREATE INDEX IF NOT EXISTS idx_concert_events_provider ON concert_events(provider);
CREATE INDEX IF NOT EXISTS idx_venues_city_country ON venues(city, country);
CREATE INDEX IF NOT EXISTS idx_concert_artists_name ON concert_artists(name);
CREATE INDEX IF NOT EXISTS idx_event_artists_event ON concert_event_artists(event_id);
CREATE INDEX IF NOT EXISTS idx_event_artists_artist ON concert_event_artists(artist_id);

-- Create updated_at trigger function if not exists
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add updated_at triggers
CREATE TRIGGER update_venues_updated_at 
    BEFORE UPDATE ON venues 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_concert_artists_updated_at 
    BEFORE UPDATE ON concert_artists 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_concert_events_updated_at 
    BEFORE UPDATE ON concert_events 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_concert_events_date_venue ON concert_events(event_date, venue_id);

-- Add comments for documentation
COMMENT ON TABLE venues IS 'Concert venues with location information';
COMMENT ON TABLE concert_artists IS 'Artists/performers separate from MusicBrainz artists table';
COMMENT ON TABLE concert_events IS 'Concert events fetched from external APIs like Ticketmaster';
COMMENT ON TABLE concert_event_artists IS 'Many-to-many relationship between events and artists';
COMMENT ON COLUMN concert_events.provider IS 'External API provider (ticketmaster, eventbrite, etc.)';
COMMENT ON COLUMN concert_events.external_url IS 'Original URL from the external provider';
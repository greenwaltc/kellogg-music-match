-- V022: User concert event interest mapping table
-- Represents a user's interest/intent regarding a specific concert event.
-- Interest states:
--   INTERESTED            - User is interested in the event
--   GOING                 - User has committed/likely attending
--   LOOKING_FOR_GROUP     - User wants to attend and is seeking others
-- Absence of a record means no expressed interest.

CREATE TABLE IF NOT EXISTS user_concert_event_interest (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id VARCHAR(255) NOT NULL REFERENCES concert_events(id) ON DELETE CASCADE,
    interest_status VARCHAR(30) NOT NULL,
    note TEXT, -- Optional user-provided note (e.g., "Need 2 more for carpool")
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, event_id),
    CONSTRAINT valid_interest_status CHECK (
        interest_status IN ('INTERESTED', 'GOING', 'LOOKING_FOR_GROUP')
    )
);

-- Maintain updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_user_concert_event_interest_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_user_concert_event_interest_updated_at
    BEFORE UPDATE ON user_concert_event_interest
    FOR EACH ROW EXECUTE FUNCTION update_user_concert_event_interest_updated_at();

-- Indexes to speed up common queries:
-- 1. Fetch all interested users for an event
CREATE INDEX IF NOT EXISTS idx_user_concert_interest_event ON user_concert_event_interest(event_id);
-- 2. Fetch all events a user has marked
CREATE INDEX IF NOT EXISTS idx_user_concert_interest_user ON user_concert_event_interest(user_id);
-- 3. Filter by status for an event (composite for event queries scoped by status)
CREATE INDEX IF NOT EXISTS idx_user_concert_interest_event_status ON user_concert_event_interest(event_id, interest_status);
-- 4. Optionally support queries filtering by status globally (status-only index)
CREATE INDEX IF NOT EXISTS idx_user_concert_interest_status ON user_concert_event_interest(interest_status);

-- Comments for documentation
COMMENT ON TABLE user_concert_event_interest IS 'User interest (INTERESTED, GOING, LOOKING_FOR_GROUP) for concert events';
COMMENT ON COLUMN user_concert_event_interest.interest_status IS 'Enumerated interest status: INTERESTED | GOING | LOOKING_FOR_GROUP';
COMMENT ON COLUMN user_concert_event_interest.note IS 'Optional free-form note by user about their attendance plans';

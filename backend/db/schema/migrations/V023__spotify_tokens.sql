-- V023: Spotify tokens storage
-- Stores OAuth tokens for Spotify per user. Refresh token is encrypted at rest by application layer.
-- Access tokens are short-lived; we may choose not to store them long-term, but keep latest for debugging.
-- A unique constraint on user_id ensures one active record; we upsert on refresh.

CREATE TABLE IF NOT EXISTS spotify_tokens (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token_encrypted BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    scope TEXT,
    token_type TEXT DEFAULT 'Bearer',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Maintain updated_at on change
CREATE OR REPLACE FUNCTION update_spotify_tokens_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_spotify_tokens_updated_at
    BEFORE UPDATE ON spotify_tokens
    FOR EACH ROW EXECUTE FUNCTION update_spotify_tokens_updated_at();

CREATE INDEX IF NOT EXISTS idx_spotify_tokens_expires_at ON spotify_tokens(expires_at);

COMMENT ON TABLE spotify_tokens IS 'Per-user Spotify OAuth tokens (refresh token encrypted at rest)';
COMMENT ON COLUMN spotify_tokens.refresh_token_encrypted IS 'Ciphertext (AES-GCM or similar) of refresh token';
COMMENT ON COLUMN spotify_tokens.scope IS 'Granted OAuth scopes';

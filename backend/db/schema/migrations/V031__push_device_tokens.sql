-- Native push device tokens (APNs/FCM)
CREATE TABLE IF NOT EXISTS push_device_tokens (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  platform TEXT NOT NULL CHECK (platform IN ('ios','android')),
  token TEXT NOT NULL,
  bundle_id TEXT,      -- for iOS, helps prevent cross-app token mixups
  app_package TEXT,    -- for Android
  device_model TEXT,
  os_version TEXT,
  app_version TEXT,
  last_seen_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(user_id, platform, token)
);

DROP TRIGGER IF EXISTS update_push_device_tokens_updated_at ON push_device_tokens;
CREATE TRIGGER update_push_device_tokens_updated_at
BEFORE UPDATE ON push_device_tokens
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_push_device_tokens_user ON push_device_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_push_device_tokens_platform ON push_device_tokens(platform);

#!/bin/sh
set -e

CONFIG_PATH="/usr/share/nginx/html/config.json"

# Gather environment (with defaults)
API_BASE_URL=${API_BASE_URL:-/api}
ARTIST_MIN_COUNT=${ARTIST_MIN_COUNT:-5}
ARTIST_MAX_COUNT=${ARTIST_MAX_COUNT:-20}
SPOTIFY_CLIENT_ID=${SPOTIFY_CLIENT_ID:-}
SPOTIFY_REDIRECT_URI=${SPOTIFY_REDIRECT_URI:-}

generate_config() {
  cat > "$CONFIG_PATH" <<EOF
{
  "apiBaseUrl": "${API_BASE_URL}",
  "artistMinCount": ${ARTIST_MIN_COUNT},
  "artistMaxCount": ${ARTIST_MAX_COUNT},
  "spotifyClientId": "${SPOTIFY_CLIENT_ID}",
  "spotifyRedirectUri": "${SPOTIFY_REDIRECT_URI}"
}
EOF
  echo "[entrypoint] Generated runtime config.json (API_BASE_URL=${API_BASE_URL})"
}

if [ -w "$CONFIG_PATH" ]; then
  generate_config
else
  echo "[entrypoint] config.json not writable (likely mounted ConfigMap); keeping existing file"
fi

exec "$@"
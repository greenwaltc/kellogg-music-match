#!/usr/bin/env bash
set -euo pipefail

# Allow overriding binary path
APP_BIN="/app/server"
DEBUG_FLAG="${DEBUG:-false}"
DELVE_BIN="/usr/local/bin/dlv"

if [[ "$DEBUG_FLAG" == "true" ]]; then
  if [[ -x "$DELVE_BIN" ]]; then
    echo "[entrypoint] Starting server under Delve (headless)"
    exec "$DELVE_BIN" exec --headless=true --listen=0.0.0.0:2345 --api-version=2 --accept-multiclient "$APP_BIN" -- "$@"
  else
    echo "[entrypoint] DEBUG=true but Delve not present in image. Starting normally." >&2
    exec "$APP_BIN" "$@"
  fi
else
  echo "[entrypoint] Starting server normally (DEBUG=false)"
  exec "$APP_BIN" "$@"
fi

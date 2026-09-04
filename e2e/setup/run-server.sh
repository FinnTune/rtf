#!/bin/bash
# Playwright's webServer.command for the E2E suite: builds and runs the real
# Go binary (serving the real webapp/dist build) against a disposable SQLite
# database, so E2E runs never touch a developer's real database/forum.db.
# Mirrors database/createDB.sh (schema creation) and the README's TLS-cert
# generation instructions, so it works unattended in CI and reuses whatever
# a local developer already has set up.
set -euo pipefail

# Resolve paths relative to this script's location, not the caller's CWD —
# Playwright runs webServer.command with CWD set to the directory containing
# playwright.config.ts (e2e/), but this script may also be run by hand.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
E2E_DIR="$REPO_ROOT/e2e"

TLS_CERT="$REPO_ROOT/localhost.crt"
TLS_KEY="$REPO_ROOT/localhost.key"
DB_DIR="$E2E_DIR/.tmp"
DB_PATH="$DB_DIR/e2e.db"
BIN_DIR="$E2E_DIR/.bin"
BIN_PATH="$BIN_DIR/rtforum"
PORT="8543"

# Reuse existing local-dev certs if present; generate fresh self-signed ones
# otherwise (always the case on a clean CI checkout — *.crt/*.key are
# gitignored). -subj avoids the interactive prompts openssl req would
# otherwise show, which would hang here unattended.
if [ ! -f "$TLS_CERT" ] || [ ! -f "$TLS_KEY" ]; then
  echo "No TLS cert/key found at repo root - generating a local self-signed one for E2E..."
  openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
    -keyout "$TLS_KEY" -out "$TLS_CERT" \
    -subj "/CN=localhost"
fi

# Always a fresh database — every E2E run starts from the same clean seeded
# state, same as a brand-new deployment.
mkdir -p "$DB_DIR"
rm -f "$DB_PATH"
sqlite3 "$DB_PATH" < "$REPO_ROOT/database/createTables.sql"

mkdir -p "$BIN_DIR"
(
  cd "$REPO_ROOT"
  CGO_ENABLED=1 CGO_CFLAGS="-Wno-discarded-qualifiers" go build -o "$BIN_PATH" .
)

# exec, not a backgrounded call — Playwright sends its stop signal to this
# process, and it needs to reach the actual server, not this wrapper script.
cd "$REPO_ROOT"
export PORT="$PORT"
export DB_PATH="$DB_PATH"
export TLS_CERT="$TLS_CERT"
export TLS_KEY="$TLS_KEY"
export ALLOWED_ORIGIN="https://localhost:$PORT"
exec "$BIN_PATH"

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

# Seeds a fixed admin account directly (bypassing registration — this isn't
# about testing registration, and it keeps auth.setup.ts's login to a
# single rate-limited call for this identity instead of two) so admin-only
# E2E flows have a real admin to log in as. migrate()'s own "promote a user
# literally named admin" convenience (see database/sqlFuncs.go) picks this
# row up and sets its role the moment the server starts below — the exact
# mechanism a real deployment's documented manual bootstrap relies on.
#
# The password hash is a precomputed bcrypt hash (cost 12) of
# "E2eAdminPassword123" (see e2e/helpers/users.ts's adminUser) — fixed and
# reused across runs rather than regenerated, since bcrypt verification
# doesn't care that the hash itself is reused. Uses a single-quoted heredoc
# delimiter so the shell never tries to expand the hash's literal $
# characters as variable references.
sqlite3 "$DB_PATH" <<'SQL'
INSERT INTO user (fname, lname, uname, email, age, gender, pass, created_at)
VALUES ('E2E', 'Admin', 'admin', 'e2e_admin@example.com', '30', 'other',
  '$2a$12$1teOdcDT5dyAbrXgZAZUKeTHn1wJVPh4M5NAp.jXncwBg2hBerTpO',
  datetime('now'));
SQL

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

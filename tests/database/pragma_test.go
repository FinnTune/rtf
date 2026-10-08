package database_test

import (
	"path/filepath"
	"strings"
	"testing"

	"rtForum/database"
)

// TestOpenDB_EnablesWALMode guards OpenDB's _journal_mode=WAL DSN option —
// the default rollback-journal mode locks the whole database file for the
// duration of any write transaction, so two users posting/reacting/sending
// a message at the same moment can genuinely collide. Opens a real
// file-backed database (not :memory:, where journal_mode is moot) through
// the actual production entry point and reads the mode back via PRAGMA,
// rather than trusting the DSN string alone — a typo or an option the
// driver silently ignores wouldn't be caught by just inspecting the DSN.
func TestOpenDB_EnablesWALMode(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	seedDB := openPlainSQLiteFile(t, dbPath)
	if _, err := seedDB.Exec(foreignKeySchema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	if err := seedDB.Close(); err != nil {
		t.Fatalf("failed to close seeding connection: %v", err)
	}

	db := database.OpenDB(dbPath)
	t.Cleanup(func() { db.Close() })

	var journalMode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("failed to read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected journal_mode %q, got %q", "wal", journalMode)
	}
}

// TestOpenDB_SetsBusyTimeout guards the _busy_timeout=5000 DSN option — WAL
// mode (above) still allows only one writer at a time, so this is the
// backstop for the write-vs-write case WAL alone doesn't remove: a second
// concurrent writer should retry for up to 5s instead of failing outright
// with SQLITE_BUSY. Pins the value explicitly rather than resting on it
// being go-sqlite3's own current default (which it already is — this test
// doesn't catch a present bug, it catches a future regression: either the
// DSN option being removed/changed, or the driver's default ever changing
// out from under an app that never set it explicitly).
func TestOpenDB_SetsBusyTimeout(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	seedDB := openPlainSQLiteFile(t, dbPath)
	if _, err := seedDB.Exec(foreignKeySchema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	if err := seedDB.Close(); err != nil {
		t.Fatalf("failed to close seeding connection: %v", err)
	}

	db := database.OpenDB(dbPath)
	t.Cleanup(func() { db.Close() })

	var busyTimeoutMs int
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeoutMs); err != nil {
		t.Fatalf("failed to read busy_timeout: %v", err)
	}
	if busyTimeoutMs != 5000 {
		t.Fatalf("expected busy_timeout 5000ms, got %dms", busyTimeoutMs)
	}
}

// TestOpenDB_IndexesNotificationAndBookmarkByPostID guards the two indexes
// that back DeletePostHandler's own cleanup (DELETE FROM notification/
// bookmark WHERE post_id = ?) — without them, deleting a post full-scans
// both tables. Unlike the WAL/busy_timeout tests above, there's no
// observable *behavior* difference an index makes (same query results
// either way, just a scan instead of a seek), so this checks the index
// actually exists (sqlite_master) and that SQLite's own query planner
// picks it for exactly the query DeletePostHandler runs, rather than just
// trusting the CREATE INDEX statement ran.
func TestOpenDB_IndexesNotificationAndBookmarkByPostID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	seedDB := openPlainSQLiteFile(t, dbPath)
	if _, err := seedDB.Exec(foreignKeySchema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	if err := seedDB.Close(); err != nil {
		t.Fatalf("failed to close seeding connection: %v", err)
	}

	db := database.OpenDB(dbPath)
	t.Cleanup(func() { db.Close() })

	for _, index := range []struct{ name, table string }{
		{"idx_notification_post_id", "notification"},
		{"idx_bookmark_post_id", "bookmark"},
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index.name).Scan(&count); err != nil {
			t.Fatalf("failed to check for index %s: %v", index.name, err)
		}
		if count != 1 {
			t.Fatalf("expected index %s to exist after migration, got %d matching sqlite_master rows", index.name, count)
		}

		var queryPlan string
		if err := db.QueryRow(`EXPLAIN QUERY PLAN DELETE FROM ` + index.table + ` WHERE post_id = 1`).Scan(new(int), new(int), new(int), &queryPlan); err != nil {
			t.Fatalf("failed to explain query plan for %s: %v", index.table, err)
		}
		if !strings.Contains(queryPlan, index.name) {
			t.Fatalf("expected DELETE FROM %s WHERE post_id = ? to use %s, got query plan: %q", index.table, index.name, queryPlan)
		}
	}
}

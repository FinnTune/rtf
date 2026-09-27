package database_test

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"rtForum/database"

	_ "github.com/mattn/go-sqlite3"
)

// openPlainSQLiteFile opens dbPath with no DSN options — foreign-key
// enforcement off, matching sql.Open's (and the sqlite3 CLI's) default —
// so this test's seeding step recreates the same starting state a real
// deployment's createDB.sh/docker-entrypoint.sh leaves a fresh database in,
// before database.OpenDB ever touches it.
func openPlainSQLiteFile(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open %s: %v", dbPath, err)
	}
	return db
}

// foreignKeySchema is a minimal schema migrate() can run against — the same
// shape preRoleSchema (migrate_test.go) uses, since migrate()'s unconditional
// steps (e.g. CREATE INDEX ... ON comment/category_relation/message) require
// those tables to already exist. Deliberately created with foreign keys
// disabled (sql.Open's default, and what a real deployment's sqlite3-CLI
// seeding step also runs with, matching createDB.sh/docker-entrypoint.sh) so
// this test starts from the same enforcement-off state OpenDB has to turn on.
const foreignKeySchema = `
CREATE TABLE user (
	id INTEGER NOT NULL PRIMARY KEY,
	fname VARCHAR(30) NOT NULL,
	lname VARCHAR(30) NOT NULL,
	uname VARCHAR(30) NOT NULL UNIQUE,
	email VARCHAR(30) NOT NULL UNIQUE,
	age VARCHAR(3) NOT NULL,
	gender VARCHAR(10) NOT NULL,
	pass VARCHAR(20) NOT NULL,
	created_at VARCHAR(30) NOT NULL
);

CREATE TABLE post (
	id INTEGER NOT NULL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	title VARCHAR(30) NOT NULL,
	content VARCHAR(150) NOT NULL,
	author VARCHAR(30) NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY(user_id) REFERENCES user(id)
);

CREATE TABLE comment (
	id INTEGER NOT NULL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	post_id INTEGER NOT NULL,
	content VARCHAR(150) NOT NULL,
	created_at DATETIME NOT NULL
);

CREATE TABLE category_relation (
	id INTEGER NOT NULL PRIMARY KEY,
	category_id INTEGER NOT NULL,
	post_id INTEGER NOT NULL
);

CREATE TABLE message (
	id INTEGER NOT NULL PRIMARY KEY,
	from_user INTEGER NOT NULL,
	to_user INTEGER NOT NULL,
	is_read TINYINT(1) NOT NULL,
	txt TEXT NOT NULL,
	created_at DATETIME NOT NULL
);
`

// TestOpenDB_EnforcesForeignKeys guards against a real, previously-unnoticed
// gap: OpenDB (the connection every production write goes through) never
// enabled SQLite's foreign-key enforcement, so every FOREIGN KEY in
// createTables.sql was purely decorative — a post insert referencing a
// nonexistent user, for instance, would silently succeed instead of
// erroring. Builds a real file-backed database (not :memory:, and not
// through a helper that already forces enforcement on) with enforcement
// left off — the same state a real deployment's sqlite3-CLI seeding step
// creates it in — then opens it through the actual production entry point
// (database.OpenDB) and confirms enforcement is active on the connection(s)
// it hands back.
func TestOpenDB_EnforcesForeignKeys(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")

	seedDB := openPlainSQLiteFile(t, dbPath)
	if _, err := seedDB.Exec(foreignKeySchema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	if _, err := seedDB.Exec(`
		INSERT INTO user (id, fname, lname, uname, email, age, gender, pass, created_at) VALUES
		(1, 'A', 'B', 'alice', 'alice@example.com', '30', 'other', 'hash', datetime('now')),
		(2, 'C', 'D', 'bob', 'bob@example.com', '31', 'other', 'hash', datetime('now'))`,
	); err != nil {
		t.Fatalf("failed to seed users: %v", err)
	}
	// A legacy-shape message row, resolvable to the two real users above —
	// this is what migrate()'s one-time conversation-model migration has to
	// turn into conversation/conversation_member/message rows, and it's the
	// first time in this codebase's history that migration has ever run
	// with foreign-key enforcement active. Its INSERTs must satisfy the new
	// conversation/message tables' own FOREIGN KEY(...) REFERENCES user(id)
	// constraints, or an already-deployed database with real chat history
	// would fail to start at all (migrate() failing is fatal — see OpenDB).
	if _, err := seedDB.Exec(
		`INSERT INTO message (from_user, to_user, is_read, txt, created_at) VALUES ('alice', 'bob', 0, 'hi bob', datetime('now'))`,
	); err != nil {
		t.Fatalf("failed to seed legacy message: %v", err)
	}
	if err := seedDB.Close(); err != nil {
		t.Fatalf("failed to close seeding connection: %v", err)
	}

	db := database.OpenDB(dbPath)
	t.Cleanup(func() { db.Close() })

	var migratedMessageCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE txt = 'hi bob'`).Scan(&migratedMessageCount); err != nil {
		t.Fatalf("failed to count migrated messages: %v", err)
	}
	if migratedMessageCount != 1 {
		t.Fatalf("expected the legacy message to have been migrated into the new message table, got %d matching rows", migratedMessageCount)
	}

	// A post referencing a real, existing user must still succeed —
	// enforcement must not be so strict it clips legitimate writes.
	if _, err := db.Exec(
		`INSERT INTO post (user_id, title, content, author, created_at) VALUES (1, 't', 'c', 'alice', datetime('now'))`,
	); err != nil {
		t.Fatalf("expected a post referencing an existing user to succeed, got: %v", err)
	}

	// A post referencing a nonexistent user must now be rejected — this is
	// the actual behavior change: before enabling _foreign_keys=on, this
	// exact insert succeeded silently.
	_, err := db.Exec(
		`INSERT INTO post (user_id, title, content, author, created_at) VALUES (999, 't2', 'c2', 'ghost', datetime('now'))`,
	)
	if err == nil {
		t.Fatal("expected inserting a post with a nonexistent user_id to fail once foreign keys are enforced")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("expected a foreign key constraint error, got: %v", err)
	}
}

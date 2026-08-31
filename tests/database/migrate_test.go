package database_test

import (
	"database/sql"
	"testing"

	"rtForum/database"

	_ "github.com/mattn/go-sqlite3"
)

// preRoleSchema is the user table shape before the role column existed —
// standing in for an already-deployed database that predates the
// migration, the exact case OpenDB's migrate() has to handle.
const preRoleSchema = `
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
`

func openPreRoleDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(preRoleSchema); err != nil {
		t.Fatalf("failed to create pre-role schema: %v", err)
	}
	return db
}

func TestMigrate_AddsRoleColumnToExistingDatabase(t *testing.T) {
	db := openPreRoleDB(t)

	hasRole, err := database.ColumnExistsForTest(db, "user", "role")
	if err != nil {
		t.Fatalf("ColumnExistsForTest: %v", err)
	}
	if hasRole {
		t.Fatal("expected the pre-migration schema to not already have a role column")
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("MigrateForTest: %v", err)
	}

	hasRole, err = database.ColumnExistsForTest(db, "user", "role")
	if err != nil {
		t.Fatalf("ColumnExistsForTest after migrate: %v", err)
	}
	if !hasRole {
		t.Fatal("expected migrate to add a role column")
	}
}

func TestMigrate_DefaultsExistingUsersToRoleUser(t *testing.T) {
	db := openPreRoleDB(t)
	if _, err := db.Exec(`INSERT INTO user (id, fname, lname, uname, email, age, gender, pass, created_at)
		VALUES (1, 'Pre', 'Existing', 'preexisting', 'pre@example.com', '30', 'other', 'hash', datetime('now'))`); err != nil {
		t.Fatalf("failed to seed pre-existing user: %v", err)
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("MigrateForTest: %v", err)
	}

	var role string
	if err := db.QueryRow(`SELECT role FROM user WHERE uname = 'preexisting'`).Scan(&role); err != nil {
		t.Fatalf("failed to query role: %v", err)
	}
	if role != "user" {
		t.Fatalf("expected a pre-existing user to default to role 'user', got %q", role)
	}
}

func TestMigrate_PromotesSeedAdminUser(t *testing.T) {
	db := openPreRoleDB(t)
	if _, err := db.Exec(`INSERT INTO user (id, fname, lname, uname, email, age, gender, pass, created_at)
		VALUES (1, 'Admin', 'User', 'admin', 'admin@example.com', '30', 'other', 'hash', datetime('now'))`); err != nil {
		t.Fatalf("failed to seed admin user: %v", err)
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("MigrateForTest: %v", err)
	}

	var role string
	if err := db.QueryRow(`SELECT role FROM user WHERE uname = 'admin'`).Scan(&role); err != nil {
		t.Fatalf("failed to query role: %v", err)
	}
	if role != "admin" {
		t.Fatalf("expected a user literally named 'admin' to be promoted, got role %q", role)
	}
}

func TestMigrate_IsIdempotent(t *testing.T) {
	db := openPreRoleDB(t)

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("first MigrateForTest: %v", err)
	}
	// A second run against an already-migrated database (the normal case on
	// every subsequent server start) must not error — ALTER TABLE ADD
	// COLUMN would fail outright if run unconditionally a second time.
	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("second MigrateForTest (idempotency check): %v", err)
	}
}

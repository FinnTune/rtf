package database_test

import (
	"database/sql"
	"testing"

	"rtForum/database"

	_ "github.com/mattn/go-sqlite3"
)

// preRoleSchema is the user table shape before the role column existed —
// standing in for an already-deployed database that predates the
// migration, the exact case OpenDB's migrate() has to handle. comment,
// category_relation, and the legacy-shape message table are included
// because they're all original, core tables/shapes present since this
// project's very first schema — any real database this old would already
// have them — unlike user_post_reaction/conversation_member, which
// migrate() itself is responsible for conditionally creating.
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

CREATE TABLE post (
	id INTEGER NOT NULL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	title VARCHAR(30) NOT NULL,
	content VARCHAR(150) NOT NULL,
	author VARCHAR(30) NOT NULL,
	created_at DATETIME NOT NULL
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

func TestMigrate_AddsImgURLColumnToExistingDatabase(t *testing.T) {
	db := openPreRoleDB(t)

	hasImgURL, err := database.ColumnExistsForTest(db, "post", "img_url")
	if err != nil {
		t.Fatalf("ColumnExistsForTest: %v", err)
	}
	if hasImgURL {
		t.Fatal("expected the pre-migration schema to not already have an img_url column")
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("MigrateForTest: %v", err)
	}

	hasImgURL, err = database.ColumnExistsForTest(db, "post", "img_url")
	if err != nil {
		t.Fatalf("ColumnExistsForTest after migrate: %v", err)
	}
	if !hasImgURL {
		t.Fatal("expected migrate to add an img_url column")
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

// openCategoryOnlyDB builds on the same pre-role schema every other test in
// this file uses (so migrate()'s earlier, unconditional steps — e.g.
// ALTER TABLE user ADD COLUMN — don't fail on a table that isn't there) and
// adds an un-migrated category table (no unique constraint on
// category_name) — the shape needed to test addCategoryNameUniqueIndex.
func openCategoryOnlyDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openPreRoleDB(t)
	if _, err := db.Exec(`CREATE TABLE category (id INTEGER NOT NULL PRIMARY KEY, category_name VARCHAR(30) NOT NULL)`); err != nil {
		t.Fatalf("failed to create category table: %v", err)
	}
	return db
}

// TestMigrate_AddsCategoryNameUniqueIndex covers the actual fix: an
// already-deployed database with CreateCategoryHandler/EditCategoryHandler's
// old check-then-act as the only thing preventing duplicate category names
// must pick up idx_category_name_unique on its next start, closing that
// TOCTOU gap at the DB layer for good.
func TestMigrate_AddsCategoryNameUniqueIndex(t *testing.T) {
	db := openCategoryOnlyDB(t)
	if _, err := db.Exec(`INSERT INTO category (category_name) VALUES ('Cuisine'), ('Places')`); err != nil {
		t.Fatalf("failed to seed categories: %v", err)
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("MigrateForTest: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO category (category_name) VALUES ('Cuisine')`); err == nil {
		t.Fatal("expected the unique index to reject a duplicate category name after migration")
	}
	// A genuinely new name must still be free to insert — the migration
	// shouldn't have broken normal category creation.
	if _, err := db.Exec(`INSERT INTO category (category_name) VALUES ('Music')`); err != nil {
		t.Fatalf("expected a new category name to still be insertable: %v", err)
	}
}

// TestMigrate_SkipsCategoryNameUniqueIndexWhenDuplicatesAlreadyExist covers
// the safety net: migrate() is fatal to server startup on error (see
// OpenDB), so a database that already has duplicate category names (from
// hitting the exact race this migration closes, before ever restarting)
// must not be permanently unable to start — it should skip adding the
// constraint (loudly) rather than failing outright.
func TestMigrate_SkipsCategoryNameUniqueIndexWhenDuplicatesAlreadyExist(t *testing.T) {
	db := openCategoryOnlyDB(t)
	if _, err := db.Exec(`INSERT INTO category (category_name) VALUES ('Cuisine'), ('Cuisine')`); err != nil {
		t.Fatalf("failed to seed duplicate categories: %v", err)
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("expected MigrateForTest to skip rather than fail on pre-existing duplicates, got: %v", err)
	}

	// Confirms the index genuinely wasn't created (rather than having
	// silently succeeded some other way) — a third 'Cuisine' must still be
	// insertable.
	if _, err := db.Exec(`INSERT INTO category (category_name) VALUES ('Cuisine')`); err != nil {
		t.Fatalf("expected no unique constraint to have been added, but insert failed: %v", err)
	}
}

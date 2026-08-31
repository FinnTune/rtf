package database

import (
	"database/sql"
	"fmt"
	"log"

	//sqlite3
	_ "github.com/mattn/go-sqlite3"
)

var ForumDB *sql.DB

func OpenDB() *sql.DB {
	dataBase, err := sql.Open("sqlite3", "./database/forum.db")
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}
	log.Println("Database opened successfully.")

	if err := migrate(dataBase); err != nil {
		log.Fatalf("Error migrating database: %s", err)
	}
	log.Println("Database migrations applied.")

	return dataBase
}

// migrate applies schema changes needed by an existing database that
// predate them — createTables.sql only runs once, on a brand-new database
// (see docker-entrypoint.sh), so anything added to it later needs an
// idempotent migration here too or an already-deployed database never picks
// it up.
func migrate(db *sql.DB) error {
	hasRole, err := columnExists(db, "user", "role")
	if err != nil {
		return fmt.Errorf("checking for user.role column: %w", err)
	}
	if !hasRole {
		if _, err := db.Exec(`ALTER TABLE user ADD COLUMN role VARCHAR(10) NOT NULL DEFAULT 'user'`); err != nil {
			return fmt.Errorf("adding user.role column: %w", err)
		}
		log.Println("Migrated: added role column to user table.")
	}

	// Convenience for local/dev databases that happen to have a real user
	// literally named "admin" (the production seed data in createTables.sql
	// does not insert one — categories/posts reference "admin" as a plain
	// author string, not a real account). Harmless no-op otherwise; this is
	// not how the first real admin should be granted in a genuine
	// deployment — see the README for the manual bootstrap step.
	if _, err := db.Exec(`UPDATE user SET role = 'admin' WHERE uname = 'admin' AND role != 'admin'`); err != nil {
		return fmt.Errorf("promoting seed admin user: %w", err)
	}

	return nil
}

// columnExists reports whether table has a column named column. table is
// always a hardcoded literal at call sites, never user input — PRAGMA
// statements don't support bound parameters, so it's interpolated directly.
func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

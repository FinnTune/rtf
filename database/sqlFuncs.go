package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	//sqlite3
	_ "github.com/mattn/go-sqlite3"
)

var ForumDB *sql.DB

func OpenDB() *sql.DB {
	dataBase, err := sql.Open("sqlite3", "./database/forum.db")
	if err != nil {
		slog.Error("error opening database", "error", err)
		os.Exit(1)
	}
	slog.Info("database opened successfully")

	if err := migrate(dataBase); err != nil {
		slog.Error("error migrating database", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")

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
		slog.Info("migrated: added role column to user table")
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

	hasImgURL, err := columnExists(db, "post", "img_url")
	if err != nil {
		return fmt.Errorf("checking for post.img_url column: %w", err)
	}
	if !hasImgURL {
		if _, err := db.Exec(`ALTER TABLE post ADD COLUMN img_url VARCHAR(200)`); err != nil {
			return fmt.Errorf("adding post.img_url column: %w", err)
		}
		slog.Info("migrated: added img_url column to post table")
	}

	// A brand-new table, unlike user.role above, so CREATE TABLE IF NOT
	// EXISTS is all the idempotency an already-deployed database needs —
	// no column-existence dance required.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_post_reaction (
			id INTEGER NOT NULL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			is_liked TINYINT(1) NOT NULL,
			created_at DATETIME NOT NULL,
			UNIQUE(user_id, post_id),
			FOREIGN KEY(user_id) REFERENCES user(id),
			FOREIGN KEY(post_id) REFERENCES post(id)
		)`); err != nil {
		return fmt.Errorf("creating user_post_reaction table: %w", err)
	}

	if err := migrateConversationModel(db); err != nil {
		return fmt.Errorf("migrating message table to conversation model: %w", err)
	}

	// A brand-new table, unlike user.role above, so CREATE TABLE IF NOT
	// EXISTS is all the idempotency an already-deployed database needs —
	// no column-existence dance required.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS message_read (
			id INTEGER NOT NULL PRIMARY KEY,
			conversation_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			last_read_message_id INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL,
			UNIQUE(conversation_id, user_id),
			FOREIGN KEY(conversation_id) REFERENCES conversation(id),
			FOREIGN KEY(user_id) REFERENCES user(id)
		)`); err != nil {
		return fmt.Errorf("creating message_read table: %w", err)
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

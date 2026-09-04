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

// OpenDB opens (and migrates) the SQLite database at path — normally
// "./database/forum.db", but overridable (see main.go's DB_PATH env var) so
// e.g. the E2E test suite can point at a disposable file instead of a
// developer's real database.
func OpenDB(path string) *sql.DB {
	dataBase, err := sql.Open("sqlite3", path)
	if err != nil {
		slog.Error("error opening database", "error", err, "path", path)
		os.Exit(1)
	}
	slog.Info("database opened successfully", "path", path)

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

	hasBanned, err := columnExists(db, "user", "banned")
	if err != nil {
		return fmt.Errorf("checking for user.banned column: %w", err)
	}
	if !hasBanned {
		if _, err := db.Exec(`ALTER TABLE user ADD COLUMN banned TINYINT(1) NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("adding user.banned column: %w", err)
		}
		slog.Info("migrated: added banned column to user table")
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

	// CREATE INDEX IF NOT EXISTS is idempotent on its own — no existence
	// check needed, unlike the ALTER TABLE column additions above. These
	// back hot-path lookups (chat history, comment listing/counts, category
	// filtering, reaction batching, per-author feeds, and every user's own
	// conversation list) that SQLite would otherwise full-scan for, since it
	// doesn't auto-index foreign-key columns.
	indexStatements := []string{
		`CREATE INDEX IF NOT EXISTS idx_message_conversation_id ON message(conversation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_post_id ON comment(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_category_relation_post_id ON category_relation(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_category_relation_category_id ON category_relation(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_post_reaction_post_id ON user_post_reaction(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_conversation_member_user_id ON conversation_member(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_post_author ON post(author)`,
	}
	for _, stmt := range indexStatements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("creating index (%s): %w", stmt, err)
		}
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

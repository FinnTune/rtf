package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	//sqlite3
	_ "github.com/mattn/go-sqlite3"
)

var ForumDB *sql.DB

// OpenDB opens (and migrates) the SQLite database at path — normally
// "./database/forum.db", but overridable (see main.go's DB_PATH env var) so
// e.g. the E2E test suite can point at a disposable file instead of a
// developer's real database.
func OpenDB(path string) *sql.DB {
	// _foreign_keys=on: SQLite disables foreign-key enforcement by default
	// on every new connection, so without this every FOREIGN KEY in
	// createTables.sql (post, comment, category_relation,
	// conversation_member, message, ...) is purely decorative — a bad
	// insert/update referencing a nonexistent row would silently succeed
	// instead of erroring. Set via the DSN (not a one-off PRAGMA exec)
	// because database/sql pools multiple underlying connections; the
	// driver applies this to every connection it opens, matching what
	// tests/testutil/database.go already does for the test schema.
	dataBase, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
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
		CREATE TABLE IF NOT EXISTS notification (
			id INTEGER NOT NULL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			comment_id INTEGER NOT NULL,
			actor_username VARCHAR(30) NOT NULL,
			created_at DATETIME NOT NULL,
			read_at DATETIME,
			FOREIGN KEY(user_id) REFERENCES user(id),
			FOREIGN KEY(post_id) REFERENCES post(id),
			FOREIGN KEY(comment_id) REFERENCES comment(id)
		)`); err != nil {
		return fmt.Errorf("creating notification table: %w", err)
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
		`CREATE INDEX IF NOT EXISTS idx_notification_user_id ON notification(user_id)`,
	}
	for _, stmt := range indexStatements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("creating index (%s): %w", stmt, err)
		}
	}

	if err := addCategoryNameUniqueIndex(db); err != nil {
		return fmt.Errorf("adding category_name unique index: %w", err)
	}

	if err := makeCategoryNameUniqueIndexCaseInsensitive(db); err != nil {
		return fmt.Errorf("making category_name unique index case-insensitive: %w", err)
	}

	if err := normalizeExistingEmailsToLowercase(db); err != nil {
		return fmt.Errorf("normalizing existing emails to lowercase: %w", err)
	}

	return nil
}

// addCategoryNameUniqueIndex closes the same TOCTOU gap already fixed for
// CreateCategoryHandler/EditCategoryHandler at the DB layer (see
// createTables.sql's idx_category_name_unique, which a brand-new database
// already gets). Unlike the CREATE INDEX IF NOT EXISTS statements above, a
// UNIQUE index can't just be added unconditionally to an already-deployed
// database: if the missing constraint has ever actually let duplicate
// category names through, creating it here would fail outright and, since
// migrate() failing is fatal (see OpenDB), take the whole server down on
// every future start until someone manually de-duplicates. Checking first
// and only skipping (loudly) in that case keeps this migration as safe as
// every other one here — a database that's never hit the race this fixes
// picks up the constraint exactly like a fresh one would.
func addCategoryNameUniqueIndex(db *sql.DB) error {
	// category is one of this app's original tables in every real
	// deployment, but migrate() also runs against deliberately minimal
	// synthetic schemas in tests (simulating a database old enough to
	// predate later columns) that don't necessarily include it — skip
	// rather than error in that case, the same way the rest of migrate()
	// only acts on tables/columns it confirms exist first.
	var hasCategoryTable int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'category'`).Scan(&hasCategoryTable); err != nil {
		return fmt.Errorf("checking for category table: %w", err)
	}
	if hasCategoryTable == 0 {
		return nil
	}

	// CREATE UNIQUE INDEX IF NOT EXISTS would still fail outright (not
	// silently skip) if duplicate category names already exist and the
	// index doesn't — this check-first is what CREATE INDEX IF NOT EXISTS
	// alone can't provide for a UNIQUE index specifically.
	var dupes int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT category_name FROM category GROUP BY category_name HAVING COUNT(*) > 1
		)`).Scan(&dupes); err != nil {
		return fmt.Errorf("checking for duplicate category names: %w", err)
	}
	if dupes > 0 {
		slog.Warn("skipping category_name unique index: duplicate category names already exist in this database", "distinct_duplicated_names", dupes)
		return nil
	}

	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_category_name_unique ON category(category_name)`); err != nil {
		return fmt.Errorf("creating unique index on category_name: %w", err)
	}
	return nil
}

// makeCategoryNameUniqueIndexCaseInsensitive upgrades idx_category_name_unique
// to compare case-insensitively (COLLATE NOCASE) — without this, "Sports"
// and "sports" pass as two distinct, independently creatable/renameable
// category names despite the index's own name suggesting otherwise. Unlike
// user.email (normalizeExistingEmailsToLowercase), category_name is a
// case-preserving display string, so this makes the COMPARISON
// case-insensitive rather than normalizing stored values to lowercase.
//
// Runs unconditionally on every start (not just once) since it's cheap and
// idempotent: the SQL-inspection check below makes it a no-op the moment
// the index is already case-insensitive, whether that's because a brand-
// new database's createTables.sql already creates it that way, or because
// this migration already ran here before.
//
// Same check-first safety as addCategoryNameUniqueIndex: replacing the
// index would fail outright if any two existing categories already differ
// only by case (a UNIQUE index can't be created over data that already
// violates it), and since migrate() failing is fatal (see OpenDB), that
// would take the whole server down on every future start. Skip (loudly) in
// that case instead, leaving the existing case-sensitive index in place.
func makeCategoryNameUniqueIndexCaseInsensitive(db *sql.DB) error {
	var hasCategoryTable int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'category'`).Scan(&hasCategoryTable); err != nil {
		return fmt.Errorf("checking for category table: %w", err)
	}
	if hasCategoryTable == 0 {
		return nil
	}

	var currentIndexSQL sql.NullString
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'idx_category_name_unique'`).Scan(&currentIndexSQL)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("checking existing category_name index: %w", err)
	}
	if err == nil && strings.Contains(strings.ToUpper(currentIndexSQL.String), "NOCASE") {
		return nil
	}

	var dupes int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT LOWER(category_name) FROM category GROUP BY LOWER(category_name) HAVING COUNT(*) > 1
		)`).Scan(&dupes); err != nil {
		return fmt.Errorf("checking for case-insensitive duplicate category names: %w", err)
	}
	if dupes > 0 {
		slog.Warn("skipping case-insensitive category_name unique index: two or more existing categories share a name differing only by case", "distinct_duplicated_names", dupes)
		return nil
	}

	// No index to drop is fine too (IF EXISTS) — that's the case where
	// addCategoryNameUniqueIndex itself skipped (exact-duplicate names
	// already existed there, a stricter condition than the case-insensitive
	// dupe check above already just cleared).
	if _, err := db.Exec(`DROP INDEX IF EXISTS idx_category_name_unique`); err != nil {
		return fmt.Errorf("dropping case-sensitive category_name index: %w", err)
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX idx_category_name_unique ON category(category_name COLLATE NOCASE)`); err != nil {
		return fmt.Errorf("creating case-insensitive unique index on category_name: %w", err)
	}
	return nil
}

// normalizeExistingEmailsToLowercase backfills user.email to lowercase for
// rows written before validateRegistration/validateProfileUpdate started
// normalizing it (see validateRegistration's doc comment for why email,
// unlike username, is treated as case-insensitive) — without this, an
// already-deployed database's older rows would stay mixed-case forever,
// and a new registration/profile-update that's now correctly rejected as
// a case-only duplicate of one of THOSE older rows would otherwise depend
// entirely on the application-level LOWER() checks (serveLogin,
// UpdateProfileHandler) rather than the data itself ever becoming
// canonical.
//
// Same check-first safety as addCategoryNameUniqueIndex: if two existing
// rows already differ only by case (impossible for rows written after
// this migration first ships, but not ruled out for older ones), blindly
// lowercasing would make both rows try to hold the identical email string,
// tripping user.email's UNIQUE constraint and crashing the server on every
// future start (migrate() failing is fatal — see OpenDB). Skipping in that
// case leaves those specific rows for manual cleanup, same as
// addCategoryNameUniqueIndex does for duplicate category names.
func normalizeExistingEmailsToLowercase(db *sql.DB) error {
	var collidingGroups int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT LOWER(email) FROM user GROUP BY LOWER(email) HAVING COUNT(*) > 1
		)`).Scan(&collidingGroups); err != nil {
		return fmt.Errorf("checking for email case collisions: %w", err)
	}
	if collidingGroups > 0 {
		slog.Warn("skipping email lowercase backfill: two or more existing accounts share an email differing only by case", "colliding_groups", collidingGroups)
		return nil
	}

	if _, err := db.Exec(`UPDATE user SET email = LOWER(email) WHERE email != LOWER(email)`); err != nil {
		return fmt.Errorf("lowercasing existing emails: %w", err)
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

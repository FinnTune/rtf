package database_test

import (
	"database/sql"
	"testing"

	"rtForum/database"
)

// seedPreConversationUsersAndMessages seeds two users onto the shared
// pre-role/pre-img_url schema from migrate_test.go, which already includes
// the legacy-shape message table (the shape createTables.sql used before
// the conversation model) — standing in for an already-deployed database
// that predates the conversation migration.
func seedPreConversationUsersAndMessages(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO user (id, fname, lname, uname, email, age, gender, pass, created_at) VALUES
		(1, 'Admin', 'User', 'admin', 'admin@example.com', '30', 'other', 'hash', datetime('now')),
		(2, 'Alice', 'Smith', 'alice', 'alice@example.com', '25', 'female', 'hash', datetime('now'));
	`); err != nil {
		t.Fatalf("failed to seed users: %v", err)
	}
}

func TestMigrate_RebuildsLegacyMessagesIntoConversations(t *testing.T) {
	db := openPreRoleDB(t)
	seedPreConversationUsersAndMessages(t, db)
	if _, err := db.Exec(`
		INSERT INTO message (from_user, to_user, is_read, txt, created_at) VALUES
		('admin', 'alice', 0, 'hello alice', datetime('now')),
		('alice', 'admin', 0, 'hi admin', datetime('now')),
		('admin', 'alice', 0, 'second message', datetime('now'));
	`); err != nil {
		t.Fatalf("failed to seed legacy messages: %v", err)
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("MigrateForTest: %v", err)
	}

	var conversationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation WHERE direct_pair_key = '1-2'`).Scan(&conversationCount); err != nil {
		t.Fatalf("failed to count conversations: %v", err)
	}
	if conversationCount != 1 {
		t.Fatalf("expected exactly one conversation for the admin/alice pair, got %d", conversationCount)
	}

	var memberCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM conversation_member
		JOIN conversation ON conversation.id = conversation_member.conversation_id
		WHERE conversation.direct_pair_key = '1-2'`).Scan(&memberCount); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCount != 2 {
		t.Fatalf("expected 2 conversation members, got %d", memberCount)
	}

	rows, err := db.Query(`
		SELECT sender_id, txt FROM message
		JOIN conversation ON conversation.id = message.conversation_id
		WHERE conversation.direct_pair_key = '1-2'
		ORDER BY message.id ASC`)
	if err != nil {
		t.Fatalf("failed to query migrated messages: %v", err)
	}
	defer rows.Close()

	type gotRow struct {
		senderID int
		txt      string
	}
	var results []gotRow
	for rows.Next() {
		var g gotRow
		if err := rows.Scan(&g.senderID, &g.txt); err != nil {
			t.Fatalf("failed to scan migrated message: %v", err)
		}
		results = append(results, g)
	}
	want := []gotRow{
		{senderID: 1, txt: "hello alice"},
		{senderID: 2, txt: "hi admin"},
		{senderID: 1, txt: "second message"},
	}
	if len(results) != len(want) {
		t.Fatalf("expected %d migrated messages, got %d: %+v", len(want), len(results), results)
	}
	for i, w := range want {
		if results[i] != w {
			t.Fatalf("message %d: got %+v, want %+v", i, results[i], w)
		}
	}

	// The pre-migration table is kept, renamed, as a safety net rather than
	// dropped.
	var legacyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message_legacy`).Scan(&legacyCount); err != nil {
		t.Fatalf("expected message_legacy to still exist: %v", err)
	}
	if legacyCount != 3 {
		t.Fatalf("expected 3 rows preserved in message_legacy, got %d", legacyCount)
	}
}

func TestMigrate_SkipsLegacyMessagesWithUnresolvableUsernames(t *testing.T) {
	db := openPreRoleDB(t)
	seedPreConversationUsersAndMessages(t, db)
	if _, err := db.Exec(`
		INSERT INTO message (from_user, to_user, is_read, txt, created_at) VALUES
		('admin', 'ghost-user', 0, 'message to a deleted user', datetime('now')),
		('admin', 'alice', 0, 'a real message', datetime('now'));
	`); err != nil {
		t.Fatalf("failed to seed legacy messages: %v", err)
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("MigrateForTest: %v", err)
	}

	var messageCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message`).Scan(&messageCount); err != nil {
		t.Fatalf("failed to count messages: %v", err)
	}
	if messageCount != 1 {
		t.Fatalf("expected only the resolvable message to be migrated, got %d", messageCount)
	}
}

func TestMigrate_ConversationMigrationIsIdempotent(t *testing.T) {
	db := openPreRoleDB(t)
	seedPreConversationUsersAndMessages(t, db)
	if _, err := db.Exec(`
		INSERT INTO message (from_user, to_user, is_read, txt, created_at) VALUES
		('admin', 'alice', 0, 'hello alice', datetime('now'));
	`); err != nil {
		t.Fatalf("failed to seed legacy messages: %v", err)
	}

	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("first MigrateForTest: %v", err)
	}
	// A second run must not re-migrate message_legacy into duplicate
	// conversations/messages — the message table no longer has a from_user
	// column after the first run, which is exactly the signal that skips
	// the migration on subsequent runs.
	if err := database.MigrateForTest(db); err != nil {
		t.Fatalf("second MigrateForTest (idempotency check): %v", err)
	}

	var messageCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message`).Scan(&messageCount); err != nil {
		t.Fatalf("failed to count messages: %v", err)
	}
	if messageCount != 1 {
		t.Fatalf("expected the second migrate run to be a no-op, got %d messages", messageCount)
	}
}

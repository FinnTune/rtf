package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// migrateConversationModel replaces the legacy message.from_user/to_user
// pairwise design (usernames stored as text in nominally-INTEGER columns)
// with a conversation/conversation_member model that can also represent
// group chats. Runs once per already-deployed database — a fresh database
// created from createTables.sql already has the new shape, so this is a
// no-op there.
func migrateConversationModel(db *sql.DB) error {
	hasLegacyColumn, err := columnExists(db, "message", "from_user")
	if err != nil {
		return fmt.Errorf("checking for message.from_user column: %w", err)
	}
	if !hasLegacyColumn {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning conversation migration transaction: %w", err)
	}
	defer tx.Rollback()

	// Renamed rather than dropped — a leftover copy of the pre-migration
	// data costs nothing and is a safety net if anything here turns out to
	// be wrong.
	if _, err := tx.Exec(`ALTER TABLE message RENAME TO message_legacy`); err != nil {
		return fmt.Errorf("renaming legacy message table: %w", err)
	}

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS conversation (
			id INTEGER NOT NULL PRIMARY KEY,
			is_group TINYINT(1) NOT NULL DEFAULT 0,
			name VARCHAR(50),
			direct_pair_key VARCHAR(21),
			created_at DATETIME NOT NULL,
			UNIQUE(direct_pair_key)
		)`); err != nil {
		return fmt.Errorf("creating conversation table: %w", err)
	}
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_member (
			id INTEGER NOT NULL PRIMARY KEY,
			conversation_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			joined_at DATETIME NOT NULL,
			UNIQUE(conversation_id, user_id),
			FOREIGN KEY(conversation_id) REFERENCES conversation(id),
			FOREIGN KEY(user_id) REFERENCES user(id)
		)`); err != nil {
		return fmt.Errorf("creating conversation_member table: %w", err)
	}
	if _, err := tx.Exec(`
		CREATE TABLE message (
			id INTEGER NOT NULL PRIMARY KEY,
			conversation_id INTEGER NOT NULL,
			sender_id INTEGER NOT NULL,
			txt TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY(conversation_id) REFERENCES conversation(id),
			FOREIGN KEY(sender_id) REFERENCES user(id)
		)`); err != nil {
		return fmt.Errorf("creating new message table: %w", err)
	}

	userIDByName, err := loadUsernameIndex(tx)
	if err != nil {
		return err
	}

	legacyMessages, err := loadLegacyMessages(tx)
	if err != nil {
		return err
	}

	migrated, skipped, err := reinsertLegacyMessages(tx, userIDByName, legacyMessages)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing conversation migration: %w", err)
	}

	log.Printf(
		"Migrated: rebuilt message table into conversation model (%d messages migrated, %d skipped for unresolvable usernames). Pre-migration data kept in message_legacy.",
		migrated, skipped,
	)
	return nil
}

func loadUsernameIndex(tx *sql.Tx) (map[string]int, error) {
	rows, err := tx.Query(`SELECT id, uname FROM user`)
	if err != nil {
		return nil, fmt.Errorf("loading users for conversation migration: %w", err)
	}
	defer rows.Close()

	userIDByName := make(map[string]int)
	for rows.Next() {
		var id int
		var uname string
		if err := rows.Scan(&id, &uname); err != nil {
			return nil, fmt.Errorf("scanning user for conversation migration: %w", err)
		}
		userIDByName[uname] = id
	}
	return userIDByName, rows.Err()
}

type legacyMessage struct {
	fromUname, toUname string
	txt, createdAt     string
}

// loadLegacyMessages reads every row out of message_legacy before any
// migration INSERTs run — go-sqlite3's driver doesn't allow another
// statement on the same transaction/connection while a Rows cursor from it
// is still open.
func loadLegacyMessages(tx *sql.Tx) ([]legacyMessage, error) {
	// CAST to TEXT normalizes the legacy columns' values regardless of
	// which storage class SQLite happened to pick for a given row — a
	// username that happens to look numeric (e.g. a test fixture using "1")
	// gets stored with INTEGER affinity, while a normal username is stored
	// as TEXT; both need to compare equal to `uname` (TEXT) below.
	rows, err := tx.Query(`
		SELECT CAST(from_user AS TEXT), CAST(to_user AS TEXT), txt, created_at
		FROM message_legacy ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("reading legacy messages: %w", err)
	}
	defer rows.Close()

	var messages []legacyMessage
	for rows.Next() {
		var lm legacyMessage
		if err := rows.Scan(&lm.fromUname, &lm.toUname, &lm.txt, &lm.createdAt); err != nil {
			return nil, fmt.Errorf("scanning legacy message: %w", err)
		}
		messages = append(messages, lm)
	}
	return messages, rows.Err()
}

// reinsertLegacyMessages replays each legacy message into the new schema,
// creating (and reusing, per unordered user-id pair) a direct conversation
// as needed. A legacy row whose from/to username no longer resolves to a
// real user — e.g. old test fixtures or a deleted account — is skipped
// rather than failing the whole migration.
func reinsertLegacyMessages(tx *sql.Tx, userIDByName map[string]int, messages []legacyMessage) (migrated, skipped int, err error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	conversationByPair := make(map[[2]int]int64)

	for _, lm := range messages {
		fromID, ok := userIDByName[lm.fromUname]
		if !ok {
			skipped++
			continue
		}
		toID, ok := userIDByName[lm.toUname]
		if !ok {
			skipped++
			continue
		}

		lo, hi := fromID, toID
		if lo > hi {
			lo, hi = hi, lo
		}
		pair := [2]int{lo, hi}
		convID, ok := conversationByPair[pair]
		if !ok {
			convID, err = createDirectConversation(tx, lo, hi, now)
			if err != nil {
				return 0, 0, err
			}
			conversationByPair[pair] = convID
		}

		if _, err := tx.Exec(
			`INSERT INTO message (conversation_id, sender_id, txt, created_at) VALUES (?, ?, ?, ?)`,
			convID, fromID, lm.txt, lm.createdAt,
		); err != nil {
			return 0, 0, fmt.Errorf("inserting migrated message: %w", err)
		}
		migrated++
	}
	return migrated, skipped, nil
}

func createDirectConversation(tx *sql.Tx, userAID, userBID int, joinedAt string) (int64, error) {
	pairKey := fmt.Sprintf("%d-%d", userAID, userBID)
	result, err := tx.Exec(`INSERT INTO conversation (is_group, direct_pair_key, created_at) VALUES (0, ?, ?)`, pairKey, joinedAt)
	if err != nil {
		return 0, fmt.Errorf("creating conversation: %w", err)
	}
	convID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("reading new conversation id: %w", err)
	}
	for _, userID := range [2]int{userAID, userBID} {
		if _, err := tx.Exec(
			`INSERT INTO conversation_member (conversation_id, user_id, joined_at) VALUES (?, ?, ?)`,
			convID, userID, joinedAt,
		); err != nil {
			return 0, fmt.Errorf("adding conversation member: %w", err)
		}
	}
	return convID, nil
}

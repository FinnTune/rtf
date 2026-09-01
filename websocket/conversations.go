package websocket

import (
	"database/sql"
	"fmt"
	"rtForum/database"
	"time"
)

// lookupUserIDByUsername resolves a username to a user id. Returns
// (0, sql.ErrNoRows) if no such user exists — callers treat that as "drop
// silently" rather than an error, matching the old from_user/to_user
// design's behavior of not hard-failing a chat action over a bad username.
func lookupUserIDByUsername(username string) (int, error) {
	var id int
	err := database.ForumDB.QueryRow(`SELECT id FROM user WHERE uname = ?`, username).Scan(&id)
	return id, err
}

// resolveOrCreateDirectConversation returns the id of the 1:1 conversation
// between two users, creating it (and its two conversation_member rows) if
// this is their first message. Race-safe under concurrent first messages
// from the same pair via conversation.direct_pair_key's UNIQUE constraint +
// INSERT OR IGNORE, rather than a check-then-create that two simultaneous
// requests could each "win".
func resolveOrCreateDirectConversation(userAID, userBID int) (int, error) {
	lo, hi := userAID, userBID
	if lo > hi {
		lo, hi = hi, lo
	}
	pairKey := fmt.Sprintf("%d-%d", lo, hi)
	now := time.Now().Format("2006-01-02 15:04:05")

	if _, err := database.ForumDB.Exec(
		`INSERT OR IGNORE INTO conversation (is_group, direct_pair_key, created_at) VALUES (0, ?, ?)`,
		pairKey, now,
	); err != nil {
		return 0, fmt.Errorf("creating direct conversation: %w", err)
	}

	var convID int
	if err := database.ForumDB.QueryRow(`SELECT id FROM conversation WHERE direct_pair_key = ?`, pairKey).Scan(&convID); err != nil {
		return 0, fmt.Errorf("looking up direct conversation: %w", err)
	}

	// Unconditional, not just on first creation — the INSERT OR IGNORE
	// above is a no-op when the conversation already existed, so this is
	// what actually guarantees both members are recorded.
	for _, uid := range [2]int{lo, hi} {
		if _, err := database.ForumDB.Exec(
			`INSERT OR IGNORE INTO conversation_member (conversation_id, user_id, joined_at) VALUES (?, ?, ?)`,
			convID, uid, now,
		); err != nil {
			return 0, fmt.Errorf("adding conversation member: %w", err)
		}
	}

	return convID, nil
}

// findDirectConversation looks up the 1:1 conversation between two users
// without creating one — used by history reads, which shouldn't have the
// side effect of creating a conversation two users have never actually
// messaged in. Returns (0, false, nil) if none exists yet.
func findDirectConversation(userAID, userBID int) (int, bool, error) {
	lo, hi := userAID, userBID
	if lo > hi {
		lo, hi = hi, lo
	}
	pairKey := fmt.Sprintf("%d-%d", lo, hi)

	var convID int
	err := database.ForumDB.QueryRow(`SELECT id FROM conversation WHERE direct_pair_key = ?`, pairKey).Scan(&convID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return convID, true, nil
}

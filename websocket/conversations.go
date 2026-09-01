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

// createGroupConversation creates a named group conversation with the given
// members (the creator's id must already be included by the caller).
func createGroupConversation(name string, memberIDs []int) (int, error) {
	now := time.Now().Format("2006-01-02 15:04:05")

	result, err := database.ForumDB.Exec(
		`INSERT INTO conversation (is_group, name, created_at) VALUES (1, ?, ?)`, name, now,
	)
	if err != nil {
		return 0, fmt.Errorf("creating group conversation: %w", err)
	}
	convID64, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("reading new group conversation id: %w", err)
	}
	convID := int(convID64)

	for _, uid := range memberIDs {
		if _, err := database.ForumDB.Exec(
			`INSERT OR IGNORE INTO conversation_member (conversation_id, user_id, joined_at) VALUES (?, ?, ?)`,
			convID, uid, now,
		); err != nil {
			return 0, fmt.Errorf("adding group member: %w", err)
		}
	}

	return convID, nil
}

// isConversationMember reports whether userID belongs to conversation
// convID — every chat action that names a conversation_id has to check
// this before doing anything with it, since the id itself is guessable and
// carries no authorization on its own.
func isConversationMember(convID, userID int) (bool, error) {
	var exists int
	err := database.ForumDB.QueryRow(
		`SELECT 1 FROM conversation_member WHERE conversation_id = ? AND user_id = ?`, convID, userID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// getConversationInfo loads a conversation's metadata (is_group, name, and
// its full member list) for the "chat-opened"/"conversations-list"
// payloads. Returns (nil, false, nil) if the conversation doesn't exist.
func getConversationInfo(convID int) (*ConversationInfo, bool, error) {
	var isGroup bool
	var name sql.NullString
	err := database.ForumDB.QueryRow(
		`SELECT is_group, name FROM conversation WHERE id = ?`, convID,
	).Scan(&isGroup, &name)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("loading conversation: %w", err)
	}

	members, err := getConversationMembers(convID)
	if err != nil {
		return nil, false, err
	}

	readStates, err := getConversationReadStates(convID)
	if err != nil {
		return nil, false, err
	}

	return &ConversationInfo{
		ConversationID: convID,
		IsGroup:        isGroup,
		Name:           name.String,
		Members:        members,
		ReadStates:     readStates,
	}, true, nil
}

// getConversationMembers returns every member of a conversation as
// (user id, username) pairs.
func getConversationMembers(convID int) ([]ConversationMember, error) {
	rows, err := database.ForumDB.Query(`
		SELECT user.id, user.uname
		FROM conversation_member
		JOIN user ON user.id = conversation_member.user_id
		WHERE conversation_member.conversation_id = ?
		ORDER BY user.uname ASC`, convID)
	if err != nil {
		return nil, fmt.Errorf("loading conversation members: %w", err)
	}
	defer rows.Close()

	members := []ConversationMember{}
	for rows.Next() {
		var m ConversationMember
		if err := rows.Scan(&m.UserID, &m.Username); err != nil {
			return nil, fmt.Errorf("scanning conversation member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// getConversationMemberIDs returns the user ids of every member of a
// conversation — the lightweight form of getConversationMembers used by the
// delivery path, which only needs to match against connected clients'
// userID, not full member metadata.
func getConversationMemberIDs(convID int) ([]int, error) {
	rows, err := database.ForumDB.Query(`SELECT user_id FROM conversation_member WHERE conversation_id = ?`, convID)
	if err != nil {
		return nil, fmt.Errorf("loading conversation member ids: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning conversation member id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// getUserConversations returns every conversation userID belongs to, most
// recently created first — the "conversations-list" response sent once
// after connecting, so a user sees their existing group chats (and any
// direct conversation they've already opened before) without having to
// rediscover them by clicking someone in the online list.
func getUserConversations(userID int) ([]ConversationInfo, error) {
	rows, err := database.ForumDB.Query(`
		SELECT conversation.id
		FROM conversation
		JOIN conversation_member ON conversation_member.conversation_id = conversation.id
		WHERE conversation_member.user_id = ?
		ORDER BY conversation.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("loading user conversations: %w", err)
	}
	var convIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning conversation id: %w", err)
		}
		convIDs = append(convIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	conversations := []ConversationInfo{}
	for _, id := range convIDs {
		info, found, err := getConversationInfo(id)
		if err != nil {
			return nil, err
		}
		if found {
			conversations = append(conversations, *info)
		}
	}
	return conversations, nil
}

// getConversationReadStates returns every member's read watermark. A member
// with no message_read row yet (never read anything) is reported at 0 via
// the LEFT JOIN + COALESCE, rather than being omitted.
func getConversationReadStates(convID int) ([]ReadState, error) {
	rows, err := database.ForumDB.Query(`
		SELECT user.id, user.uname, COALESCE(message_read.last_read_message_id, 0)
		FROM conversation_member
		JOIN user ON user.id = conversation_member.user_id
		LEFT JOIN message_read ON message_read.conversation_id = conversation_member.conversation_id
			AND message_read.user_id = conversation_member.user_id
		WHERE conversation_member.conversation_id = ?
		ORDER BY user.uname ASC`, convID)
	if err != nil {
		return nil, fmt.Errorf("loading read states: %w", err)
	}
	defer rows.Close()

	states := []ReadState{}
	for rows.Next() {
		var s ReadState
		if err := rows.Scan(&s.UserID, &s.Username, &s.LastReadMessageID); err != nil {
			return nil, fmt.Errorf("scanning read state: %w", err)
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

// messageExistsInConversation reports whether messageID is a real message
// belonging to convID — mark-read validates against this so a client can't
// claim to have read a message that isn't even part of the conversation.
func messageExistsInConversation(convID, messageID int) (bool, error) {
	var exists int
	err := database.ForumDB.QueryRow(
		`SELECT 1 FROM message WHERE id = ? AND conversation_id = ?`, messageID, convID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// markConversationRead advances userID's read watermark for convID to
// messageID — but only forward, never backward, so an out-of-order
// mark-read (e.g. from a stale request) can't regress it.
func markConversationRead(convID, userID, messageID int) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := database.ForumDB.Exec(`
		INSERT INTO message_read (conversation_id, user_id, last_read_message_id, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(conversation_id, user_id) DO UPDATE SET
			last_read_message_id = MAX(last_read_message_id, excluded.last_read_message_id),
			updated_at = excluded.updated_at`,
		convID, userID, messageID, now,
	)
	if err != nil {
		return fmt.Errorf("marking conversation read: %w", err)
	}
	return nil
}

package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

func TestSendMessage_StoresInDatabase(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	// sendMessage resolves the recipient username against the real `user`
	// table now (to find/create their direct conversation), so these need
	// to be real seeded users, not placeholder identities.
	sender := websocket.AddTestClient("s1", "admin", 1)
	_ = websocket.AddTestClient("s2", "alice", 2)

	payload, _ := json.Marshal(map[string]string{
		"message": "test chat message",
		"from":    "admin",
		"to":      "alice",
	})

	if err := websocket.SendMessageForTest(payload, sender); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	var text string
	err := db.QueryRow(`SELECT txt FROM message WHERE txt = ?`, "test chat message").Scan(&text)
	if err != nil {
		t.Fatalf("message not stored in database: %v", err)
	}
}

func TestSendMessage_UsesAuthenticatedSenderNotPayloadFrom(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	// What's under test is that the payload's claimed "from" (a spoof
	// attempt) is discarded in favor of the authenticated sender's real
	// user id, regardless of what value it holds.
	sender := websocket.AddTestClient("s1", "admin", 1)

	payload, _ := json.Marshal(map[string]string{
		"message": "spoof test message",
		"from":    "root",
		"to":      "alice",
	})

	if err := websocket.SendMessageForTest(payload, sender); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	var senderID int
	err := db.QueryRow(`SELECT sender_id FROM message WHERE txt = ?`, "spoof test message").Scan(&senderID)
	if err != nil {
		t.Fatalf("message not stored in database: %v", err)
	}
	if senderID != 1 {
		t.Fatalf("message was spoofed: stored sender_id %d, want the authenticated sender's id %d", senderID, 1)
	}
}

func TestSendMessage_DropsMessageToNonexistentRecipient(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)

	payload, _ := json.Marshal(map[string]string{
		"message": "message to nobody",
		"from":    "admin",
		"to":      "does-not-exist",
	})

	// A recipient username that doesn't resolve to a real user is dropped,
	// not an error — returning an error here would kill the sender's whole
	// WebSocket connection (see routeEvent), a much harsher failure than
	// silently not storing an unsendable message.
	if err := websocket.SendMessageForTest(payload, sender); err != nil {
		t.Fatalf("sendMessage should not error for an unresolvable recipient, got: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE txt = ?`, "message to nobody").Scan(&count); err != nil {
		t.Fatalf("failed to query message count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no message stored for a nonexistent recipient, found %d", count)
	}
}

func TestSendMessage_ReusesSameConversationAcrossMessages(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)

	for _, text := range []string{"first message", "second message"} {
		payload, _ := json.Marshal(map[string]string{"message": text, "from": "admin", "to": "actual_user"})
		if err := websocket.SendMessageForTest(payload, sender); err != nil {
			t.Fatalf("sendMessage failed: %v", err)
		}
	}

	var conversationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation WHERE direct_pair_key = '1-42'`).Scan(&conversationCount); err != nil {
		t.Fatalf("failed to count conversations: %v", err)
	}
	if conversationCount != 1 {
		t.Fatalf("expected exactly one direct conversation to be created for the pair, got %d", conversationCount)
	}

	var messageCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM message
		JOIN conversation ON conversation.id = message.conversation_id
		WHERE conversation.direct_pair_key = '1-42'`).Scan(&messageCount); err != nil {
		t.Fatalf("failed to count messages: %v", err)
	}
	if messageCount != 2 {
		t.Fatalf("expected both messages to land in the same conversation, got %d", messageCount)
	}
}

func TestGetChatHistory_ReturnsEmptyForNonexistentOtherUser(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	requester := websocket.AddTestClient("s1", "admin", 1)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "admin",
		ToUser:   "does-not-exist",
		Limit:    10,
		Offset:   0,
	})

	if err := websocket.GetChatHistoryForTest(payload, requester); err != nil {
		t.Fatalf("getChatHistory failed: %v", err)
	}

	eventType, eventPayload, ok := requester.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat history")
	}
	if eventType != websocket.SendChatHistory {
		t.Fatalf("expected chat_history event, got %q", eventType)
	}
	var messages []websocket.ChatMessage
	if err := json.Unmarshal(eventPayload, &messages); err != nil {
		t.Fatalf("failed to decode chat history: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected no messages for a nonexistent other user, got %d", len(messages))
	}
}

func TestGetChatHistory_IgnoresSpoofedFromUser(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	// Seed a private conversation between two users (admin=1, actual_user=42)
	// the requester is not part of.
	convResult, err := db.Exec(`INSERT INTO conversation (is_group, direct_pair_key, created_at) VALUES (0, '1-42', datetime('now'))`)
	if err != nil {
		t.Fatalf("failed to seed conversation: %v", err)
	}
	convID, _ := convResult.LastInsertId()
	if _, err := db.Exec(
		`INSERT INTO conversation_member (conversation_id, user_id, joined_at) VALUES (?, 1, datetime('now')), (?, 42, datetime('now'))`,
		convID, convID,
	); err != nil {
		t.Fatalf("failed to seed conversation members: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO message (conversation_id, sender_id, txt, created_at) VALUES (?, 1, 'private admin-actual_user message', datetime('now'))`,
		convID,
	); err != nil {
		t.Fatalf("failed to seed message: %v", err)
	}

	// alice (real id 2) is really connected as "alice", but the request
	// claims FromUser is "admin" to try to read admin's conversation with
	// actual_user.
	requester := websocket.AddTestClient("s1", "alice", 2)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "admin",
		ToUser:   "actual_user",
		Limit:    10,
		Offset:   0,
	})

	if err := websocket.GetChatHistoryForTest(payload, requester); err != nil {
		t.Fatalf("getChatHistory failed: %v", err)
	}

	eventType, eventPayload, ok := requester.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat history")
	}
	if eventType != websocket.SendChatHistory {
		t.Fatalf("expected chat_history event, got %q", eventType)
	}

	var messages []websocket.ChatMessage
	if err := json.Unmarshal(eventPayload, &messages); err != nil {
		t.Fatalf("failed to decode chat history: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("spoofed FromUser leaked another conversation: got %d messages, want 0: %+v", len(messages), messages)
	}
}

func TestTyping_UsesAuthenticatedSenderNotPayloadFrom(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "alice", 2)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "root", // spoofed
		ToUser:   "alice",
	})

	if err := websocket.TypingForTest(payload, sender); err != nil {
		t.Fatalf("typing failed: %v", err)
	}

	_, eventPayload, ok := recipient.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for typing notification")
	}
	var forwarded websocket.ChatMessage
	if err := json.Unmarshal(eventPayload, &forwarded); err != nil {
		t.Fatalf("failed to decode typing event: %v", err)
	}
	if forwarded.FromUser != "admin" {
		t.Fatalf("typing indicator was spoofed: forwarded from %q, want the authenticated sender %q", forwarded.FromUser, "admin")
	}
}

func TestStopTyping_UsesAuthenticatedSenderNotPayloadFrom(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "alice", 2)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "root", // spoofed
		ToUser:   "alice",
	})

	if err := websocket.StopTypingForTest(payload, sender); err != nil {
		t.Fatalf("stopTyping failed: %v", err)
	}

	_, eventPayload, ok := recipient.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for stop-typing notification")
	}
	var forwarded websocket.ChatMessage
	if err := json.Unmarshal(eventPayload, &forwarded); err != nil {
		t.Fatalf("failed to decode stop-typing event: %v", err)
	}
	if forwarded.FromUser != "admin" {
		t.Fatalf("stop-typing indicator was spoofed: forwarded from %q, want the authenticated sender %q", forwarded.FromUser, "admin")
	}
}

func TestAddUserInfo_MarksAlreadyIdentifiedClientOnline(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// The client's identity is set once, at login (AddTestClient stands in
	// for newAuthenticatedClient here) — user-connect's job is only to mark
	// it online and broadcast, never to (re)establish who it is.
	client := websocket.AddTestClient("s1", "admin", 1)

	payload, _ := json.Marshal(websocket.UserSession{
		Username: "admin",
		UserID:   1,
		Email:    "admin@example.com",
		Joined:   "2024-01-01",
	})

	if err := websocket.AddUserInfoForTest(payload, client); err != nil {
		t.Fatalf("addUserInfo failed: %v", err)
	}

	if !websocket.IsInLoggedInList("admin") {
		t.Fatal("expected admin in LoggedInList")
	}

	eventType, _, ok := client.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for users-online broadcast")
	}
	if eventType != websocket.UsersList {
		t.Fatalf("expected users-online event, got %q", eventType)
	}
}

func TestAddUserInfo_IgnoresSpoofedIdentityInPayload(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// This client's real, server-verified identity is "admin"/1 (set the
	// same way newAuthenticatedClient sets it at login). A malicious or
	// non-browser client could still send a user-connect event claiming to
	// be anyone — that must never change who the server thinks this
	// connection is, since every other handler (and every HTTP request
	// sharing this session) trusts c.username/c.userID as authoritative.
	client := websocket.AddTestClient("s1", "admin", 1)

	spoofedPayload, _ := json.Marshal(websocket.UserSession{
		Username: "alice",
		UserID:   2,
		Email:    "alice@example.com",
		Joined:   "2024-01-01",
	})

	if err := websocket.AddUserInfoForTest(spoofedPayload, client); err != nil {
		t.Fatalf("addUserInfo failed: %v", err)
	}

	if client.Username() != "admin" {
		t.Fatalf("identity was spoofed via user-connect payload: expected admin, got %q", client.Username())
	}
	if client.UserID() != 1 {
		t.Fatalf("identity was spoofed via user-connect payload: expected user id 1, got %d", client.UserID())
	}
	if websocket.IsInLoggedInList("alice") {
		t.Fatal("spoofed username alice should never appear in LoggedInList")
	}
	if !websocket.IsInLoggedInList("admin") {
		t.Fatal("expected the real identity (admin) in LoggedInList")
	}
}

func TestGetChatHistory_ReturnsMessages(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// Uses the base seed's direct conversation between admin (1) and alice
	// (2), which already has two messages.
	requester := websocket.AddTestClient("s1", "admin", 1)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "admin",
		ToUser:   "alice",
		Limit:    10,
		Offset:   0,
	})

	if err := websocket.GetChatHistoryForTest(payload, requester); err != nil {
		t.Fatalf("getChatHistory failed: %v", err)
	}

	eventType, eventPayload, ok := requester.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat history")
	}
	if eventType != websocket.SendChatHistory {
		t.Fatalf("expected chat_history event, got %q", eventType)
	}

	var messages []websocket.ChatMessage
	if err := json.Unmarshal(eventPayload, &messages); err != nil {
		t.Fatalf("failed to decode chat history: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages in history, got %d", len(messages))
	}
}

func TestTyping_ForwardsToRecipient(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "alice", 2)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "admin",
		ToUser:   "alice",
		Text:     "typing...",
	})

	if err := websocket.TypingForTest(payload, sender); err != nil {
		t.Fatalf("typing failed: %v", err)
	}

	eventType, _, ok := recipient.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for typing notification")
	}
	if eventType != websocket.Typing {
		t.Fatalf("expected typing event, got %q", eventType)
	}
}

func TestStopTyping_ForwardsToRecipient(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "alice", 2)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "admin",
		ToUser:   "alice",
	})

	if err := websocket.StopTypingForTest(payload, sender); err != nil {
		t.Fatalf("stopTyping failed: %v", err)
	}

	eventType, _, ok := recipient.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for stop-typing notification")
	}
	if eventType != websocket.StopTyping {
		t.Fatalf("expected stop-typing event, got %q", eventType)
	}
}

func TestRouteEvent_UnknownType(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	client := websocket.AddTestClient("s1", "admin", 1)
	payload := json.RawMessage(`{}`)

	if err := websocket.RouteEventForTest("unknown-event", payload, client); err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

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

	// Client identity is the string form of the real user id (1 = admin) —
	// message.from_user/to_user are (loosely) foreign keys to user.id, and
	// sendMessage now always stores the authenticated sender's identity, not
	// whatever the payload's "from" field claims.
	sender := websocket.AddTestClient("s1", "1", 1)
	_ = websocket.AddTestClient("s2", "2", 2)

	payload, _ := json.Marshal(map[string]string{
		"message": "test chat message",
		"from":    "1",
		"to":      "2",
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

	// The message table's from_user/to_user columns are (loosely) foreign
	// keys to user.id, so — matching the existing test convention in
	// TestSendMessage_StoresInDatabase above — the client's username here is
	// the string form of their real user id (1 = admin). What's under test
	// is that the payload's claimed "from" (a spoof attempt) is discarded in
	// favor of the authenticated sender, regardless of what value it holds.
	sender := websocket.AddTestClient("s1", "1", 1)

	payload, _ := json.Marshal(map[string]string{
		"message": "spoof test message",
		"from":    "root",
		"to":      "2",
	})

	if err := websocket.SendMessageForTest(payload, sender); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	var fromUser string
	err := db.QueryRow(`SELECT from_user FROM message WHERE txt = ?`, "spoof test message").Scan(&fromUser)
	if err != nil {
		t.Fatalf("message not stored in database: %v", err)
	}
	if fromUser != "1" {
		t.Fatalf("message was spoofed: stored from_user %q, want the authenticated sender %q", fromUser, "1")
	}
}

func TestGetChatHistory_IgnoresSpoofedFromUser(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	// Seed a private conversation between two users (admin=1, actual_user=42)
	// the requester is not part of.
	_, err := db.Exec(`INSERT INTO message (from_user, to_user, is_read, txt, created_at) VALUES
		('1', '42', 0, 'private admin-actual_user message', datetime('now'))`)
	if err != nil {
		t.Fatalf("failed to seed message: %v", err)
	}

	// alice (real id 2) is really connected as "2", but the request claims
	// FromUser is "1" (admin) to try to read admin's conversation with
	// actual_user.
	requester := websocket.AddTestClient("s1", "2", 2)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "1",
		ToUser:   "42",
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

	requester := websocket.AddTestClient("s1", "1", 1)

	payload, _ := json.Marshal(websocket.ChatMessage{
		FromUser: "1",
		ToUser:   "2",
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

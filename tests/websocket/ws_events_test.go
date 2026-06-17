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

	sender := websocket.AddTestClient("s1", "admin", 1)
	_ = websocket.AddTestClient("s2", "alice", 2)

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

func TestAddUserInfo_UpdatesClientAndBroadcasts(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	client := websocket.AddTestClient("s1", "", 0)

	payload, _ := json.Marshal(websocket.UserSession{
		Username: "alice",
		UserID:   2,
		Email:    "alice@example.com",
		Joined:   "2024-01-01",
	})

	if err := websocket.AddUserInfoForTest(payload, client); err != nil {
		t.Fatalf("addUserInfo failed: %v", err)
	}

	if client.Username() != "alice" {
		t.Fatalf("expected username alice, got %q", client.Username())
	}
	if client.UserID() != 2 {
		t.Fatalf("expected user id 2, got %d", client.UserID())
	}
	if !websocket.IsInLoggedInList("alice") {
		t.Fatal("expected alice in LoggedInList")
	}

	eventType, _, ok := client.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for users-online broadcast")
	}
	if eventType != websocket.UsersList {
		t.Fatalf("expected users-online event, got %q", eventType)
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

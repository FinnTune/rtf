package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

func markReadPayload(t *testing.T, convID, messageID int) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(websocket.MarkReadRequest{ConversationID: convID, MessageID: messageID})
	if err != nil {
		t.Fatalf("failed to marshal mark-read payload: %v", err)
	}
	return payload
}

func TestSendMessage_AutoMarksSenderOwnMessageRead(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, sender, "actual_user")

	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, "hello"), sender); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	var messageID, watermark int
	if err := db.QueryRow(`SELECT id FROM message WHERE txt = ?`, "hello").Scan(&messageID); err != nil {
		t.Fatalf("failed to find the sent message: %v", err)
	}
	if err := db.QueryRow(
		`SELECT last_read_message_id FROM message_read WHERE conversation_id = ? AND user_id = 1`, info.ConversationID,
	).Scan(&watermark); err != nil {
		t.Fatalf("failed to read sender's watermark: %v", err)
	}
	if watermark != messageID {
		t.Fatalf("expected sender's watermark to auto-advance to %d, got %d", messageID, watermark)
	}
}

func TestMarkRead_AdvancesWatermarkAndBroadcastsReceipt(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "actual_user", 42)
	// mustOpenDirectChat already consumes sender's chat-opened response.
	info := mustOpenDirectChat(t, sender, "actual_user")

	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, "hi"), sender); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}
	sender.WaitEvent(time.Second)    // drain the sender's message-ack
	recipient.WaitEvent(time.Second) // drain sent-message

	var messageID int
	if err := db.QueryRow(`SELECT id FROM message WHERE txt = ?`, "hi").Scan(&messageID); err != nil {
		t.Fatalf("failed to find the sent message: %v", err)
	}

	if err := websocket.MarkReadForTest(markReadPayload(t, info.ConversationID, messageID), recipient); err != nil {
		t.Fatalf("markRead failed: %v", err)
	}

	var watermark int
	if err := db.QueryRow(
		`SELECT last_read_message_id FROM message_read WHERE conversation_id = ? AND user_id = 42`, info.ConversationID,
	).Scan(&watermark); err != nil {
		t.Fatalf("failed to read recipient's watermark: %v", err)
	}
	if watermark != messageID {
		t.Fatalf("expected watermark %d, got %d", messageID, watermark)
	}

	eventType, eventPayload, ok := sender.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for read-receipt")
	}
	if eventType != websocket.ReadReceipt {
		t.Fatalf("expected read-receipt event, got %q", eventType)
	}
	var receipt websocket.ReadReceiptEvent
	if err := json.Unmarshal(eventPayload, &receipt); err != nil {
		t.Fatalf("failed to decode read-receipt: %v", err)
	}
	if receipt.Username != "actual_user" || receipt.MessageID != messageID || receipt.ConversationID != info.ConversationID {
		t.Fatalf("unexpected read-receipt payload: %+v", receipt)
	}

	// The reader themselves shouldn't get an echo of their own receipt.
	if _, _, ok := recipient.WaitEvent(100 * time.Millisecond); ok {
		t.Fatal("the reader should not receive their own read-receipt back")
	}
}

func TestMarkRead_DoesNotRegressWatermark(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "actual_user", 42)
	info := mustOpenDirectChat(t, sender, "actual_user")

	for _, text := range []string{"first", "second"} {
		if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, text), sender); err != nil {
			t.Fatalf("sendMessage failed: %v", err)
		}
	}
	var firstID, secondID int
	if err := db.QueryRow(`SELECT id FROM message WHERE txt = 'first'`).Scan(&firstID); err != nil {
		t.Fatalf("failed to find first message: %v", err)
	}
	if err := db.QueryRow(`SELECT id FROM message WHERE txt = 'second'`).Scan(&secondID); err != nil {
		t.Fatalf("failed to find second message: %v", err)
	}

	if err := websocket.MarkReadForTest(markReadPayload(t, info.ConversationID, secondID), recipient); err != nil {
		t.Fatalf("markRead (second) failed: %v", err)
	}
	// An older/out-of-order mark-read must not move the watermark backward.
	if err := websocket.MarkReadForTest(markReadPayload(t, info.ConversationID, firstID), recipient); err != nil {
		t.Fatalf("markRead (first, stale) failed: %v", err)
	}

	var watermark int
	if err := db.QueryRow(
		`SELECT last_read_message_id FROM message_read WHERE conversation_id = ? AND user_id = 42`, info.ConversationID,
	).Scan(&watermark); err != nil {
		t.Fatalf("failed to read watermark: %v", err)
	}
	if watermark != secondID {
		t.Fatalf("expected watermark to stay at %d (the higher value), got %d", secondID, watermark)
	}
}

func TestMarkRead_RejectsNonMember(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	owner := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, owner, "actual_user")
	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, "private"), owner); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}
	var messageID int
	if err := db.QueryRow(`SELECT id FROM message WHERE txt = 'private'`).Scan(&messageID); err != nil {
		t.Fatalf("failed to find message: %v", err)
	}

	outsider := websocket.AddTestClient("s2", "alice", 2)
	if err := websocket.MarkReadForTest(markReadPayload(t, info.ConversationID, messageID), outsider); err != nil {
		t.Fatalf("markRead should not error for a non-member, got: %v", err)
	}

	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM message_read WHERE conversation_id = ? AND user_id = 2`, info.ConversationID,
	).Scan(&count); err != nil {
		t.Fatalf("failed to query watermark: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no watermark recorded for a non-member, found %d", count)
	}
}

func TestMarkRead_RejectsMessageNotInConversation(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	// Two separate direct conversations: admin<->actual_user and admin<->alice.
	client := websocket.AddTestClient("s1", "admin", 1)
	otherConv := mustOpenDirectChat(t, client, "actual_user")
	if err := websocket.SendMessageForTest(sendMessagePayload(t, otherConv.ConversationID, "elsewhere"), client); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}
	var otherMessageID int
	if err := db.QueryRow(`SELECT id FROM message WHERE txt = 'elsewhere'`).Scan(&otherMessageID); err != nil {
		t.Fatalf("failed to find message: %v", err)
	}

	// The base seed's conversation id 1 is admin<->alice — try to mark it
	// read using a message id that actually belongs to the other conversation.
	if err := websocket.MarkReadForTest(markReadPayload(t, 1, otherMessageID), client); err != nil {
		t.Fatalf("markRead should not error, got: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message_read WHERE conversation_id = 1 AND user_id = 1`).Scan(&count); err != nil {
		t.Fatalf("failed to query watermark: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no watermark recorded for a message id from a different conversation, found %d", count)
	}
}

func TestOpenDirectChat_IncludesReadStates(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	requester := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, requester, "actual_user")

	if len(info.ReadStates) != 2 {
		t.Fatalf("expected read states for both members, got %d: %+v", len(info.ReadStates), info.ReadStates)
	}
	for _, state := range info.ReadStates {
		if state.LastReadMessageID != 0 {
			t.Fatalf("expected a brand-new conversation's read states to start at 0, got %+v", state)
		}
	}
}

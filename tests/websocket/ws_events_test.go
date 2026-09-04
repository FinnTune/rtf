package websocket_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

// mustOpenDirectChat drives the open-direct-chat request/response round
// trip and returns the resulting conversation info, failing the test on any
// error or unexpected event.
func mustOpenDirectChat(t *testing.T, requester *websocket.TestClientHandle, username string) websocket.ConversationInfo {
	t.Helper()
	payload, _ := json.Marshal(websocket.OpenDirectChatRequest{Username: username})
	if err := websocket.OpenDirectChatForTest(payload, requester); err != nil {
		t.Fatalf("openDirectChat failed: %v", err)
	}
	eventType, eventPayload, ok := requester.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-opened")
	}
	if eventType != websocket.ChatOpened {
		t.Fatalf("expected chat-opened event, got %q (payload %s)", eventType, eventPayload)
	}
	var info websocket.ConversationInfo
	if err := json.Unmarshal(eventPayload, &info); err != nil {
		t.Fatalf("failed to decode chat-opened: %v", err)
	}
	return info
}

func sendMessagePayload(t *testing.T, convID int, message string) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(websocket.ReceiveMessageEvent{ConversationID: convID, Message: message})
	if err != nil {
		t.Fatalf("failed to marshal send-message payload: %v", err)
	}
	return payload
}

func TestOpenDirectChat_CreatesAndReturnsConversation(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// actual_user (42) has never messaged admin (1) before — this is the
	// seed data's only pair without a pre-existing direct conversation.
	requester := websocket.AddTestClient("s1", "admin", 1)

	info := mustOpenDirectChat(t, requester, "actual_user")

	if info.IsGroup {
		t.Fatal("expected a direct (non-group) conversation")
	}
	if len(info.Members) != 2 {
		t.Fatalf("expected 2 members, got %d: %+v", len(info.Members), info.Members)
	}
}

func TestOpenDirectChat_ReusesExistingConversation(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	requester := websocket.AddTestClient("s1", "admin", 1)

	first := mustOpenDirectChat(t, requester, "actual_user")
	second := mustOpenDirectChat(t, requester, "actual_user")

	if first.ConversationID != second.ConversationID {
		t.Fatalf("expected the same conversation both times, got %d and %d", first.ConversationID, second.ConversationID)
	}
}

func TestOpenDirectChat_RejectsNonexistentUser(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	requester := websocket.AddTestClient("s1", "admin", 1)
	payload, _ := json.Marshal(websocket.OpenDirectChatRequest{Username: "does-not-exist"})

	if err := websocket.OpenDirectChatForTest(payload, requester); err != nil {
		t.Fatalf("openDirectChat should not error for an unknown user, got: %v", err)
	}

	eventType, _, ok := requester.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestOpenDirectChat_RejectsSelf(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	requester := websocket.AddTestClient("s1", "admin", 1)
	payload, _ := json.Marshal(websocket.OpenDirectChatRequest{Username: "admin"})

	if err := websocket.OpenDirectChatForTest(payload, requester); err != nil {
		t.Fatalf("openDirectChat should not error, got: %v", err)
	}

	eventType, _, ok := requester.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestSendMessage_StoresInDatabase(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, sender, "actual_user")

	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, "test chat message"), sender); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	var text string
	err := db.QueryRow(`SELECT txt FROM message WHERE txt = ?`, "test chat message").Scan(&text)
	if err != nil {
		t.Fatalf("message not stored in database: %v", err)
	}
}

func TestSendMessage_AlwaysAttributesToAuthenticatedSender(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	// The wire protocol carries no client-supplied "from"/sender field at
	// all anymore (unlike the old username-based design) — this locks in
	// that a message is always stored under the authenticated connection's
	// real user id, regardless of what extra fields a malicious payload
	// might smuggle in.
	sender := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, sender, "actual_user")

	payload, _ := json.Marshal(map[string]any{
		"conversation_id": info.ConversationID,
		"message":         "spoof test message",
		"sender_id":       999,
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

func TestSendMessage_RejectsNonMember(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	// Seed conversation 1 (admin/alice, from the base fixture) — alice is
	// authenticated as herself but tries to post into a conversation seeded
	// separately between admin and actual_user.
	other := websocket.AddTestClient("s-other", "admin", 1)
	info := mustOpenDirectChat(t, other, "actual_user")

	outsider := websocket.AddTestClient("s2", "alice", 2)
	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, "sneaky message"), outsider); err != nil {
		t.Fatalf("sendMessage should not error for a non-member, got: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE txt = ?`, "sneaky message").Scan(&count); err != nil {
		t.Fatalf("failed to query message count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no message stored for a non-member, found %d", count)
	}
}

func TestSendMessage_RejectsTooLongMessageWithoutKillingConnection(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, sender, "actual_user")

	tooLong := strings.Repeat("a", 1001)
	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, tooLong), sender); err != nil {
		t.Fatalf("sendMessage should not error (never fatal to the connection) for an over-length message, got: %v", err)
	}

	eventType, _, ok := sender.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE conversation_id = ?`, info.ConversationID).Scan(&count); err != nil {
		t.Fatalf("failed to query message count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no message stored for an over-length message, found %d", count)
	}
}

func TestSendMessage_RejectsBlankMessage(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, sender, "actual_user")

	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, "   "), sender); err != nil {
		t.Fatalf("sendMessage should not error for a blank message, got: %v", err)
	}

	eventType, _, ok := sender.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestSendMessage_AllowsMessageAtTheLengthLimit(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, sender, "actual_user")

	atLimit := strings.Repeat("a", 1000)
	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, atLimit), sender); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE txt = ?`, atLimit).Scan(&count); err != nil {
		t.Fatalf("failed to query message count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the exactly-at-limit message to be stored, found %d", count)
	}
}

func TestSendMessage_BroadcastsToOtherMemberNotSender(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "actual_user", 42)
	info := mustOpenDirectChat(t, sender, "actual_user")
	// Drain the chat-opened event sender received above.
	sender.WaitEvent(time.Second)

	if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, "hi there"), sender); err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	eventType, eventPayload, ok := recipient.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for the recipient's sent-message event")
	}
	if eventType != websocket.EventSendMessage {
		t.Fatalf("expected sent-message event, got %q", eventType)
	}
	var msg websocket.SendMessageEvent
	if err := json.Unmarshal(eventPayload, &msg); err != nil {
		t.Fatalf("failed to decode sent-message: %v", err)
	}
	if msg.From != "admin" || msg.Message != "hi there" {
		t.Fatalf("unexpected sent-message payload: %+v", msg)
	}

	// The sender gets a targeted "message-ack" (so their client learns the
	// message's real id, e.g. for read-receipt comparisons) but never a
	// "sent-message" echo of their own message.
	senderEventType, _, ok := sender.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for the sender's message-ack event")
	}
	if senderEventType != websocket.MessageAck {
		t.Fatalf("expected message-ack event, got %q", senderEventType)
	}
	if _, _, ok := sender.WaitEvent(100 * time.Millisecond); ok {
		t.Fatal("sender should not receive anything beyond the message-ack")
	}
}

func TestSendMessage_ReusesSameConversationAcrossMessages(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	info := mustOpenDirectChat(t, sender, "actual_user")

	for _, text := range []string{"first message", "second message"} {
		if err := websocket.SendMessageForTest(sendMessagePayload(t, info.ConversationID, text), sender); err != nil {
			t.Fatalf("sendMessage failed: %v", err)
		}
	}

	var messageCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM message WHERE conversation_id = ?`, info.ConversationID).Scan(&messageCount); err != nil {
		t.Fatalf("failed to count messages: %v", err)
	}
	if messageCount != 2 {
		t.Fatalf("expected both messages to land in the same conversation, got %d", messageCount)
	}
}

func TestGetChatHistory_RejectsNonMember(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	owner := websocket.AddTestClient("s-owner", "admin", 1)
	info := mustOpenDirectChat(t, owner, "actual_user")

	outsider := websocket.AddTestClient("s2", "alice", 2)
	payload, _ := json.Marshal(websocket.GetHistoryRequest{ConversationID: info.ConversationID, Limit: 10, Offset: 0})

	if err := websocket.GetChatHistoryForTest(payload, outsider); err != nil {
		t.Fatalf("getChatHistory failed: %v", err)
	}

	eventType, eventPayload, ok := outsider.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat history")
	}
	if eventType != websocket.SendChatHistory {
		t.Fatalf("expected chat_history event, got %q", eventType)
	}

	var messages []websocket.ChatHistoryMessage
	if err := json.Unmarshal(eventPayload, &messages); err != nil {
		t.Fatalf("failed to decode chat history: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("a non-member should never see another conversation's history: got %d messages, want 0: %+v", len(messages), messages)
	}
}

func TestGetChatHistory_ReturnsMessages(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// Uses the base seed's direct conversation between admin (1) and alice
	// (2) (conversation id 1), which already has two messages.
	requester := websocket.AddTestClient("s1", "admin", 1)

	payload, _ := json.Marshal(websocket.GetHistoryRequest{ConversationID: 1, Limit: 10, Offset: 0})

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

	var messages []websocket.ChatHistoryMessage
	if err := json.Unmarshal(eventPayload, &messages); err != nil {
		t.Fatalf("failed to decode chat history: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages in history, got %d", len(messages))
	}
	if messages[0].From != "admin" || messages[1].From != "alice" {
		t.Fatalf("unexpected sender attribution: %+v", messages)
	}
}

func TestTyping_ForwardsToOtherMemberNotSender(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "alice", 2)

	// Uses the base seed's direct conversation (id 1) between admin and alice.
	payload, _ := json.Marshal(websocket.TypingEvent{ConversationID: 1, From: "root"}) // From is spoofed, must be ignored

	if err := websocket.TypingForTest(payload, sender); err != nil {
		t.Fatalf("typing failed: %v", err)
	}

	eventType, eventPayload, ok := recipient.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for typing notification")
	}
	if eventType != websocket.Typing {
		t.Fatalf("expected typing event, got %q", eventType)
	}
	var forwarded websocket.TypingEvent
	if err := json.Unmarshal(eventPayload, &forwarded); err != nil {
		t.Fatalf("failed to decode typing event: %v", err)
	}
	if forwarded.From != "admin" {
		t.Fatalf("typing indicator was spoofed: forwarded from %q, want the authenticated sender %q", forwarded.From, "admin")
	}
}

func TestTyping_DoesNotForwardForNonMember(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// actual_user (42) isn't a member of conversation 1 (admin/alice).
	outsider := websocket.AddTestClient("s1", "actual_user", 42)
	recipient := websocket.AddTestClient("s2", "alice", 2)

	payload, _ := json.Marshal(websocket.TypingEvent{ConversationID: 1})
	if err := websocket.TypingForTest(payload, outsider); err != nil {
		t.Fatalf("typing should not error for a non-member, got: %v", err)
	}

	if _, _, ok := recipient.WaitEvent(200 * time.Millisecond); ok {
		t.Fatal("a non-member's typing event should never be forwarded")
	}
}

func TestStopTyping_ForwardsToOtherMemberNotSender(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	sender := websocket.AddTestClient("s1", "admin", 1)
	recipient := websocket.AddTestClient("s2", "alice", 2)

	payload, _ := json.Marshal(websocket.TypingEvent{ConversationID: 1, From: "root"})

	if err := websocket.StopTypingForTest(payload, sender); err != nil {
		t.Fatalf("stopTyping failed: %v", err)
	}

	eventType, eventPayload, ok := recipient.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for stop-typing notification")
	}
	if eventType != websocket.StopTyping {
		t.Fatalf("expected stop-typing event, got %q", eventType)
	}
	var forwarded websocket.TypingEvent
	if err := json.Unmarshal(eventPayload, &forwarded); err != nil {
		t.Fatalf("failed to decode stop-typing event: %v", err)
	}
	if forwarded.From != "admin" {
		t.Fatalf("stop-typing indicator was spoofed: forwarded from %q, want the authenticated sender %q", forwarded.From, "admin")
	}
}

func TestCreateGroupChat_CreatesConversationAndNotifiesOnlineMembers(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	member := websocket.AddTestClient("s2", "alice", 2)

	payload, _ := json.Marshal(websocket.CreateGroupChatRequest{Name: "Trip Planning", Usernames: []string{"alice", "actual_user"}})
	if err := websocket.CreateGroupChatForTest(payload, creator); err != nil {
		t.Fatalf("createGroupChat failed: %v", err)
	}

	eventType, eventPayload, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-opened (creator)")
	}
	if eventType != websocket.ChatOpened {
		t.Fatalf("expected chat-opened event, got %q", eventType)
	}
	var info websocket.ConversationInfo
	if err := json.Unmarshal(eventPayload, &info); err != nil {
		t.Fatalf("failed to decode chat-opened: %v", err)
	}
	if !info.IsGroup {
		t.Fatal("expected a group conversation")
	}
	if info.Name != "Trip Planning" {
		t.Fatalf("expected name %q, got %q", "Trip Planning", info.Name)
	}
	if len(info.Members) != 3 {
		t.Fatalf("expected 3 members (creator + 2 named), got %d: %+v", len(info.Members), info.Members)
	}

	// alice is online, so she should also get a chat-opened push immediately.
	memberEventType, _, ok := member.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-opened (online member)")
	}
	if memberEventType != websocket.ChatOpened {
		t.Fatalf("expected chat-opened event for the online member, got %q", memberEventType)
	}

	var memberCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_member WHERE conversation_id = ?`, info.ConversationID).Scan(&memberCount); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCount != 3 {
		t.Fatalf("expected 3 persisted members, got %d", memberCount)
	}
}

func TestCreateGroupChat_RejectsUnknownMember(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	payload, _ := json.Marshal(websocket.CreateGroupChatRequest{Name: "Ghosts", Usernames: []string{"does-not-exist"}})

	if err := websocket.CreateGroupChatForTest(payload, creator); err != nil {
		t.Fatalf("createGroupChat should not error, got: %v", err)
	}

	eventType, _, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestCreateGroupChat_RejectsEmptyName(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	payload, _ := json.Marshal(websocket.CreateGroupChatRequest{Name: "  ", Usernames: []string{"alice"}})

	if err := websocket.CreateGroupChatForTest(payload, creator); err != nil {
		t.Fatalf("createGroupChat should not error, got: %v", err)
	}

	eventType, _, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestCreateGroupChat_RejectsNoMembers(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	creator := websocket.AddTestClient("s1", "admin", 1)
	payload, _ := json.Marshal(websocket.CreateGroupChatRequest{Name: "Solo", Usernames: []string{}})

	if err := websocket.CreateGroupChatForTest(payload, creator); err != nil {
		t.Fatalf("createGroupChat should not error, got: %v", err)
	}

	eventType, _, ok := creator.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for chat-error")
	}
	if eventType != websocket.ChatError {
		t.Fatalf("expected chat-error event, got %q", eventType)
	}
}

func TestGetConversations_ReturnsAllUserConversations(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// admin (1) is already a member of the base seed's direct conversation
	// (id 1, with alice). Create a group too, then confirm both show up.
	requester := websocket.AddTestClient("s1", "admin", 1)
	groupPayload, _ := json.Marshal(websocket.CreateGroupChatRequest{Name: "Everyone", Usernames: []string{"alice", "actual_user"}})
	if err := websocket.CreateGroupChatForTest(groupPayload, requester); err != nil {
		t.Fatalf("createGroupChat failed: %v", err)
	}
	requester.WaitEvent(time.Second) // drain the chat-opened event

	if err := websocket.GetConversationsForTest(requester); err != nil {
		t.Fatalf("getConversations failed: %v", err)
	}

	eventType, eventPayload, ok := requester.WaitEvent(time.Second)
	if !ok {
		t.Fatal("timed out waiting for conversations-list")
	}
	if eventType != websocket.ConversationsList {
		t.Fatalf("expected conversations-list event, got %q", eventType)
	}
	var conversations []websocket.ConversationInfo
	if err := json.Unmarshal(eventPayload, &conversations); err != nil {
		t.Fatalf("failed to decode conversations-list: %v", err)
	}
	if len(conversations) != 2 {
		t.Fatalf("expected 2 conversations (1 direct + 1 group), got %d: %+v", len(conversations), conversations)
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

func TestRouteEvent_UnknownType(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	client := websocket.AddTestClient("s1", "admin", 1)
	payload := json.RawMessage(`{}`)

	if err := websocket.RouteEventForTest("unknown-event", payload, client); err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

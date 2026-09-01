package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"rtForum/database"
	"sync"
	"time"
)

type Manager struct {
	clients ClientsMapList
	sync.RWMutex
	eventHandlers map[string]EventHandler
	otps          *otpsMap
}

// Factory function for manager
func newManager(ctx context.Context) *Manager {
	log.Println("Manager created.")
	m := &Manager{
		clients:       make(ClientsMapList),
		eventHandlers: make(map[string]EventHandler),
		otps:          newOtpsMap(ctx, 5*time.Second),
	}

	//Register event handlers
	m.RegisterEventHandlers()

	return m
}

// clientsSnapshot returns a point-in-time copy of the connected clients,
// safe to range over without holding the manager's lock. Delivery loops
// need this rather than locking around the send itself: egress channels are
// unbuffered, so sending while holding m's lock would block every other
// manager operation (login, logout, ...) on however long one slow/stuck
// client's writeMesssage goroutine takes to drain it.
func (m *Manager) clientsSnapshot() []*Client {
	m.RLock()
	defer m.RUnlock()
	snapshot := make([]*Client, 0, len(m.clients))
	for c := range m.clients {
		snapshot = append(snapshot, c)
	}
	return snapshot
}

// Send message handler function
func sendMessage(event Event, c *Client) error {
	log.Printf("Event/message sent: %s", event)
	var chatEvent ReceiveMessageEvent
	if err := json.Unmarshal(event.Payload, &chatEvent); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}

	isMember, err := isConversationMember(chatEvent.ConversationID, c.userID)
	if err != nil {
		return fmt.Errorf("checking conversation membership: %s", err)
	}
	if !isMember {
		// Not a hard error — the conversation_id is client-supplied and
		// could be stale (e.g. the conversation was never opened by this
		// client) or malicious; dropping it silently avoids killing the
		// connection over what's usually just a UI race, not an attack.
		log.Printf("sendMessage: %s is not a member of conversation %d, dropping message", c.username, chatEvent.ConversationID)
		return nil
	}

	sent := time.Now()
	// Store message in sqlite3 database
	result, err := database.ForumDB.Exec(
		"INSERT INTO message (conversation_id, sender_id, txt, created_at) VALUES (?, ?, ?, ?)",
		chatEvent.ConversationID, c.userID, chatEvent.Message, sent,
	)
	if err != nil {
		return fmt.Errorf("failed to store message in database: %s", err)
	}
	messageID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to read new message id: %s", err)
	}

	// The sender has, by definition, already read their own message —
	// advance their watermark now rather than requiring a separate
	// mark-read round trip for something that's already true.
	if err := markConversationRead(chatEvent.ConversationID, c.userID, int(messageID)); err != nil {
		return fmt.Errorf("marking sender's own message read: %s", err)
	}

	chatMessage := SendMessageEvent{
		Id:             int(messageID),
		ConversationID: chatEvent.ConversationID,
		// From is always the authenticated sender, never anything the
		// client could supply — otherwise any connection could send
		// messages that appear (and are permanently stored) as coming from
		// an arbitrary other user.
		From:    c.username,
		Message: chatEvent.Message,
		Sent:    sent,
	}
	data, err := json.Marshal(chatMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message error: %s", err)
	}

	if err := broadcastToConversation(c.manager, chatEvent.ConversationID, c.userID, Event{Type: EventSendMessage, Payload: data}); err != nil {
		return err
	}

	// "sent-message" only ever reaches the conversation's OTHER members
	// (broadcastToConversation excludes the sender) — without a separate ack
	// back to the sender's own connection, they'd never learn their own
	// message's real database id and could never see a "seen by" indicator
	// advance past their own latest message.
	ackData, err := json.Marshal(MessageAckEvent{ConversationID: chatEvent.ConversationID, Id: int(messageID)})
	if err != nil {
		return fmt.Errorf("failed to marshal message-ack event: %s", err)
	}
	c.egress <- Event{Type: MessageAck, Payload: ackData}

	return nil
}

// broadcastToConversation delivers outgoingEvent to every currently
// connected client that belongs to conversation convID, except the one
// with excludeUserID (the sender, who already has the message locally).
func broadcastToConversation(m *Manager, convID, excludeUserID int, outgoingEvent Event) error {
	memberIDs, err := getConversationMemberIDs(convID)
	if err != nil {
		return fmt.Errorf("loading conversation members for broadcast: %w", err)
	}
	members := make(map[int]bool, len(memberIDs))
	for _, id := range memberIDs {
		members[id] = true
	}

	for _, recipient := range m.clientsSnapshot() {
		if recipient.userID != excludeUserID && members[recipient.userID] {
			recipient.egress <- outgoingEvent
		}
	}
	return nil
}

// addUserInfo handles the user-connect event, marking an already-identified
// client as online. It deliberately ignores any identity fields in
// event.Payload — c.username/c.userID/c.email/c.joined were already bound
// to the verified login at newAuthenticatedClient time, and trusting
// client-supplied values here would let any authenticated connection
// declare itself to be any other user.
func addUserInfo(event Event, c *Client) error {
	log.Printf("Adding user info: %s", event)

	LoggedInList.Add(c.username)
	log.Println("User:", c.username, "added to LoggedInList")

	data, err := json.Marshal(LoggedInList.Snapshot())
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message error: %s", err)
	}
	outgoingEvent := Event{
		Payload: json.RawMessage(data),
		Type:    UsersList,
	}

	for _, recipient := range c.manager.clientsSnapshot() {
		recipient.egress <- outgoingEvent
	}

	return nil
}

func getChatHistory(event Event, c *Client) error {

	var req GetHistoryRequest
	if err := json.Unmarshal(event.Payload, &req); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}
	log.Println("History Request: ", req)

	messages := []ChatHistoryMessage{}

	// Never trust a client-supplied conversation_id without checking
	// membership — otherwise any connection could read any conversation's
	// history just by guessing/incrementing an id.
	isMember, err := isConversationMember(req.ConversationID, c.userID)
	if err != nil {
		return fmt.Errorf("checking conversation membership: %s", err)
	}
	if !isMember {
		return sendChatHistory(c, messages)
	}

	members, err := getConversationMembers(req.ConversationID)
	if err != nil {
		return fmt.Errorf("loading conversation members: %s", err)
	}
	usernameByID := make(map[int]string, len(members))
	for _, m := range members {
		usernameByID[m.UserID] = m.Username
	}

	// First get the total number of rows for the specific chat conversation
	var count int
	if err := database.ForumDB.QueryRow("SELECT COUNT(*) FROM message WHERE conversation_id = ?", req.ConversationID).Scan(&count); err != nil {
		return fmt.Errorf("failed to count table lines: %s", err)
	}

	// Check if the requested offset plus limit is greater than the total number of rows
	if req.Offset+req.Limit > count {
		req.Limit = count - req.Offset // Adjust the limit
	}

	rows, err := database.ForumDB.Query(`
    SELECT * FROM (
        SELECT id, sender_id, txt, created_at FROM message
        WHERE conversation_id = ?
        ORDER BY id DESC
        LIMIT ? OFFSET ?
    ) sub
    ORDER BY id ASC`, req.ConversationID, req.Limit, req.Offset)
	if err != nil {
		return fmt.Errorf("failed to retrieve history: %s", err)

	}
	defer rows.Close()

	for rows.Next() {
		var id, senderID int
		var text, createdAt string
		if err := rows.Scan(&id, &senderID, &text, &createdAt); err != nil {
			return fmt.Errorf("failed to scan history: %s", err)
		}
		messages = append(messages, ChatHistoryMessage{
			Id:             id,
			ConversationID: req.ConversationID,
			From:           usernameByID[senderID],
			Message:        text,
			CreatedAt:      createdAt,
		})
	}
	log.Println("History of Messages: ", messages)

	return sendChatHistory(c, messages)
}

func sendChatHistory(c *Client, messages []ChatHistoryMessage) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal history message error: %s", err)
	}
	outgoingEvent := Event{
		Payload: data,
		Type:    SendChatHistory,
	}

	// Deliver directly to the requesting connection — it's already known
	// and authenticated, no need to search for it by (spoofable) username.
	c.egress <- outgoingEvent
	log.Println("History sent to: ", c.username)
	return nil
}

func typing(event Event, c *Client) error {
	return forwardTypingEvent(Typing, event, c)
}

func stopTyping(event Event, c *Client) error {
	return forwardTypingEvent(StopTyping, event, c)
}

// forwardTypingEvent decodes a "typing"/"stop-typing" payload and, once
// confirmed the sender actually belongs to the named conversation,
// broadcasts it (with From forced to the authenticated sender) to that
// conversation's other members.
func forwardTypingEvent(eventType string, event Event, c *Client) error {
	var typingEvent TypingEvent
	if err := json.Unmarshal(event.Payload, &typingEvent); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}

	isMember, err := isConversationMember(typingEvent.ConversationID, c.userID)
	if err != nil {
		return fmt.Errorf("checking conversation membership: %s", err)
	}
	if !isMember {
		return nil
	}

	// Never trust who the client claims is typing — otherwise any
	// connection could impersonate another user's typing indicator.
	typingEvent.From = c.username

	data, err := json.Marshal(typingEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal typing event: %s", err)
	}

	return broadcastToConversation(c.manager, typingEvent.ConversationID, c.userID, Event{Type: eventType, Payload: data})
}

// openDirectChat resolves (creating if needed) the 1:1 conversation between
// the requester and a named user, and replies with its metadata so the
// client can open a window keyed by conversation_id.
func openDirectChat(event Event, c *Client) error {
	var req OpenDirectChatRequest
	if err := json.Unmarshal(event.Payload, &req); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}

	otherUserID, err := lookupUserIDByUsername(req.Username)
	if err != nil {
		return sendChatError(c, fmt.Sprintf("user %q not found", req.Username))
	}
	if otherUserID == c.userID {
		return sendChatError(c, "cannot open a chat with yourself")
	}

	convID, err := resolveOrCreateDirectConversation(c.userID, otherUserID)
	if err != nil {
		return fmt.Errorf("resolving conversation: %s", err)
	}

	info, found, err := getConversationInfo(convID)
	if err != nil {
		return fmt.Errorf("loading conversation info: %s", err)
	}
	if !found {
		return fmt.Errorf("conversation %d vanished immediately after creation", convID)
	}

	return sendChatOpened(c, *info)
}

// createGroupChat creates a named group conversation containing the
// requester plus every named username, and — since a brand-new group has no
// message yet to organically notify anyone — pushes "chat-opened" to every
// named member who's currently connected, so it shows up immediately rather
// than only the next time they reconnect.
func createGroupChat(event Event, c *Client) error {
	var req CreateGroupChatRequest
	if err := json.Unmarshal(event.Payload, &req); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}

	name, usernames, err := validateGroupChat(req.Name, req.Usernames)
	if err != nil {
		return sendChatError(c, err.Error())
	}

	memberIDs := []int{c.userID}
	for _, username := range usernames {
		userID, err := lookupUserIDByUsername(username)
		if err != nil {
			return sendChatError(c, fmt.Sprintf("user %q not found", username))
		}
		if userID != c.userID {
			memberIDs = append(memberIDs, userID)
		}
	}

	convID, err := createGroupConversation(name, memberIDs)
	if err != nil {
		return fmt.Errorf("creating group conversation: %s", err)
	}

	info, found, err := getConversationInfo(convID)
	if err != nil {
		return fmt.Errorf("loading conversation info: %s", err)
	}
	if !found {
		return fmt.Errorf("conversation %d vanished immediately after creation", convID)
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal chat-opened event: %s", err)
	}
	outgoingEvent := Event{Type: ChatOpened, Payload: data}

	memberSet := make(map[int]bool, len(memberIDs))
	for _, id := range memberIDs {
		memberSet[id] = true
	}
	for _, recipient := range c.manager.clientsSnapshot() {
		if memberSet[recipient.userID] {
			recipient.egress <- outgoingEvent
		}
	}
	return nil
}

// getConversations replies with every conversation the requester belongs
// to — sent once right after connecting so existing group chats (which,
// unlike a direct chat, can't be rediscovered just by clicking an online
// user) show up without any action from the user.
func getConversations(event Event, c *Client) error {
	conversations, err := getUserConversations(c.userID)
	if err != nil {
		return fmt.Errorf("loading conversations: %s", err)
	}
	data, err := json.Marshal(conversations)
	if err != nil {
		return fmt.Errorf("failed to marshal conversations-list event: %s", err)
	}
	c.egress <- Event{Type: ConversationsList, Payload: data}
	return nil
}

// markRead handles the "mark-read" event: the client reports having seen
// everything up to and including a given message. Advances the requester's
// own watermark and broadcasts the new state to the conversation's other
// members as a "read-receipt".
func markRead(event Event, c *Client) error {
	var req MarkReadRequest
	if err := json.Unmarshal(event.Payload, &req); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}

	isMember, err := isConversationMember(req.ConversationID, c.userID)
	if err != nil {
		return fmt.Errorf("checking conversation membership: %s", err)
	}
	if !isMember {
		return nil
	}

	exists, err := messageExistsInConversation(req.ConversationID, req.MessageID)
	if err != nil {
		return fmt.Errorf("checking message existence: %s", err)
	}
	if !exists {
		// Client-supplied message_id doesn't belong to this conversation —
		// dropped rather than trusted, same posture as every other
		// client-supplied id in this package.
		return nil
	}

	if err := markConversationRead(req.ConversationID, c.userID, req.MessageID); err != nil {
		return fmt.Errorf("marking conversation read: %s", err)
	}

	receipt := ReadReceiptEvent{
		ConversationID: req.ConversationID,
		UserID:         c.userID,
		Username:       c.username,
		MessageID:      req.MessageID,
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("failed to marshal read-receipt event: %s", err)
	}

	return broadcastToConversation(c.manager, req.ConversationID, c.userID, Event{Type: ReadReceipt, Payload: data})
}

func sendChatOpened(c *Client, info ConversationInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal chat-opened event: %s", err)
	}
	c.egress <- Event{Type: ChatOpened, Payload: data}
	return nil
}

// sendChatError delivers a user-facing error to just the requesting
// connection — never broadcast, and never fatal to the connection itself
// (routeEvent only kills the connection on a non-nil Go error, and this
// always returns nil after sending).
func sendChatError(c *Client, message string) error {
	data, err := json.Marshal(ChatErrorEvent{Message: message})
	if err != nil {
		return fmt.Errorf("failed to marshal chat-error event: %s", err)
	}
	c.egress <- Event{Type: ChatError, Payload: data}
	return nil
}

func (m *Manager) RegisterEventHandlers() {
	m.eventHandlers[EventReceiveMessage] = sendMessage
	m.eventHandlers[UserConnect] = addUserInfo
	m.eventHandlers[GetChatHistory] = getChatHistory
	m.eventHandlers[GetMoreChatHistory] = getChatHistory
	m.eventHandlers[Typing] = typing
	m.eventHandlers[StopTyping] = stopTyping
	m.eventHandlers[OpenDirectChat] = openDirectChat
	m.eventHandlers[CreateGroupChat] = createGroupChat
	m.eventHandlers[GetConversations] = getConversations
	m.eventHandlers[MarkRead] = markRead

}

func (m *Manager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.eventHandlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		//Two different ways to return error
		// return fmt.Errorf("no handler for event type: %s", event.Type)
		return errors.New("no handler for event type: " + event.Type)
	}
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true //Add client to manager
	if conn := client.getConnection(); conn != nil {
		log.Println("Client:", conn.RemoteAddr(), "added to manager.")
	}
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok { //Checko if client exists in manager
		client.closeConnection()
		delete(m.clients, client)
	}
}

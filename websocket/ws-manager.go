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
	var chatMessage SendMessageEvent
	chatMessage.Sent = time.Now()
	chatMessage.Message = chatEvent.Message
	// From is always the authenticated sender, never the client-supplied
	// value — otherwise any connection could send messages that appear (and
	// are permanently stored) as coming from an arbitrary other user.
	chatMessage.From = c.username
	chatMessage.To = chatEvent.To

	// A nonexistent recipient username is dropped rather than treated as a
	// hard error — an error here would kill the sender's whole WebSocket
	// connection (see routeEvent), which is a much harsher failure than the
	// old behavior (silently storing a message to a nonexistent user).
	toUserID, err := lookupUserIDByUsername(chatEvent.To)
	if err != nil {
		log.Printf("sendMessage: recipient %q not found, dropping message: %s", chatEvent.To, err)
		return nil
	}

	convID, err := resolveOrCreateDirectConversation(c.userID, toUserID)
	if err != nil {
		return fmt.Errorf("resolving conversation: %s", err)
	}

	// Store message in sqlite3 database
	_, err = database.ForumDB.Exec("INSERT INTO message (conversation_id, sender_id, txt, created_at) VALUES (?, ?, ?, ?)",
		convID, c.userID, chatMessage.Message, chatMessage.Sent)
	if err != nil {
		return fmt.Errorf("failed to store message in database: %s", err)
	}

	data, err := json.Marshal(chatMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message error: %s", err)
	}
	outgoingEvent := Event{
		Payload: data,
		Type:    EventSendMessage,
	}

	for _, recipient := range c.manager.clientsSnapshot() {
		if recipient.username == chatEvent.To {
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

	var chtMsg ChatMessage
	if err := json.Unmarshal(event.Payload, &chtMsg); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}
	// FromUser is always the authenticated requester, never whatever the
	// client claims — otherwise any connection could request another pair's
	// private conversation just by naming it in the payload.
	chtMsg.FromUser = c.username
	log.Println("History Request: ", chtMsg)

	messages := []ChatMessage{}

	otherUserID, err := lookupUserIDByUsername(chtMsg.ToUser)
	if err != nil {
		// No such user — nothing to return, same as the old design's
		// behavior when a bogus ToUser matched zero rows.
		return sendChatHistory(c, messages)
	}

	convID, found, err := findDirectConversation(c.userID, otherUserID)
	if err != nil {
		return fmt.Errorf("looking up conversation: %s", err)
	}
	if !found {
		// The two users have never exchanged a message, so there's no
		// conversation row yet — an empty history, not an error.
		return sendChatHistory(c, messages)
	}

	// First get the total number of rows for the specific chat conversation
	var count int
	if err := database.ForumDB.QueryRow("SELECT COUNT(*) FROM message WHERE conversation_id = ?", convID).Scan(&count); err != nil {
		return fmt.Errorf("failed to count table lines: %s", err)
	}

	// Check if the requested offset plus limit is greater than the total number of rows
	if chtMsg.Offset+chtMsg.Limit > count {
		chtMsg.Limit = count - chtMsg.Offset // Adjust the limit
	}

	rows, err := database.ForumDB.Query(`
    SELECT * FROM (
        SELECT id, sender_id, txt, created_at FROM message
        WHERE conversation_id = ?
        ORDER BY id DESC
        LIMIT ? OFFSET ?
    ) sub
    ORDER BY id ASC`, convID, chtMsg.Limit, chtMsg.Offset)
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
		msg := ChatMessage{
			Id:        id,
			Text:      text,
			CreatedAt: createdAt,
			// Read receipts aren't wired up yet — every history entry
			// reports unread until that lands.
			IsRead: false,
		}
		// This is a direct (1:1) conversation between exactly the requester
		// and chtMsg.ToUser, so the sender is always one of those two —
		// resolving From/To from the already-known usernames avoids a join
		// back to `user` per row.
		if senderID == c.userID {
			msg.FromUser = c.username
			msg.ToUser = chtMsg.ToUser
		} else {
			msg.FromUser = chtMsg.ToUser
			msg.ToUser = c.username
		}
		messages = append(messages, msg)
	}
	log.Println("History of Messages: ", messages)

	return sendChatHistory(c, messages)
}

func sendChatHistory(c *Client, messages []ChatMessage) error {
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

	var chtMsg ChatMessage
	if err := json.Unmarshal(event.Payload, &chtMsg); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}
	// Never trust who the client claims is typing — otherwise any
	// connection could impersonate another user's typing indicator.
	chtMsg.FromUser = c.username

	data, err := json.Marshal(chtMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal history message error: %s", err)
	}

	outgoingEvent := Event{
		Type:    Typing,
		Payload: json.RawMessage(data),
	}

	for _, recipient := range c.manager.clientsSnapshot() {
		if recipient.username == chtMsg.ToUser {
			recipient.egress <- outgoingEvent
			log.Println("History sent to: ", recipient.username)
		}
	}
	return nil
}

func stopTyping(event Event, c *Client) error {
	var chtMsg ChatMessage
	if err := json.Unmarshal(event.Payload, &chtMsg); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}
	// Same rationale as typing() above.
	chtMsg.FromUser = c.username

	data, err := json.Marshal(chtMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal history message error: %s", err)
	}

	outgoingEvent := Event{
		Type:    StopTyping,
		Payload: json.RawMessage(data),
	}

	for _, recipient := range c.manager.clientsSnapshot() {
		if recipient.username == chtMsg.ToUser {
			recipient.egress <- outgoingEvent
			log.Println("History sent to: ", recipient.username)
		}
	}
	return nil
}

func (m *Manager) RegisterEventHandlers() {
	m.eventHandlers[EventReceiveMessage] = sendMessage
	m.eventHandlers[UserConnect] = addUserInfo
	m.eventHandlers[GetChatHistory] = getChatHistory
	m.eventHandlers[GetMoreChatHistory] = getChatHistory
	m.eventHandlers[Typing] = typing
	m.eventHandlers[StopTyping] = stopTyping

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

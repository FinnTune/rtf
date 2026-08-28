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

	// Store message in sqlite3 database
	_, err := database.ForumDB.Exec("INSERT INTO message (from_user, to_user, is_read, txt, created_at) VALUES (?, ?, ?, ?, ?)",
		chatMessage.From, chatEvent.To, 0, chatMessage.Message, chatMessage.Sent)
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
	// client claims — otherwise any connection could request (and, via the
	// old by-username delivery below, potentially have delivered) any two
	// arbitrary users' private conversation.
	chtMsg.FromUser = c.username
	log.Println("History Request: ", chtMsg)

	// First get the total number of rows for the specific chat conversation
	var count int
	err := database.ForumDB.QueryRow("SELECT COUNT(*) FROM message WHERE (from_user = ? AND to_user = ?) OR (from_user = ? AND to_user = ?)", chtMsg.FromUser, chtMsg.ToUser, chtMsg.ToUser, chtMsg.FromUser).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count table lines: %s", err)
	}

	// Check if the requested offset plus limit is greater than the total number of rows
	if chtMsg.Offset+chtMsg.Limit > count {
		chtMsg.Limit = count - chtMsg.Offset // Adjust the limit
	}

	rows, err := database.ForumDB.Query(`
    SELECT * FROM (
        SELECT * FROM message 
        WHERE (from_user = ? AND to_user = ?) OR (from_user = ? AND to_user = ?) 
        ORDER BY id DESC 
        LIMIT ? OFFSET ?
    ) sub 
    ORDER BY id ASC`, chtMsg.FromUser, chtMsg.ToUser, chtMsg.ToUser, chtMsg.FromUser, chtMsg.Limit, chtMsg.Offset)
	if err != nil {
		return fmt.Errorf("failed to retrieve history: %s", err)

	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		err = rows.Scan(&msg.Id, &msg.FromUser, &msg.ToUser, &msg.IsRead, &msg.Text, &msg.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan history: %s", err)
		}
		messages = append(messages, msg)
	}
	log.Println("History of Messages: ", messages)

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

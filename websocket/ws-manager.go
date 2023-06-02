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
	otps          otpsMap
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
	chatMessage.From = chatEvent.From
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

	for c := range c.manager.clients {
		if c.username == chatEvent.To {
			c.egress <- outgoingEvent
		}
	}

	return nil
}

func addUserInfo(event Event, c *Client) error {
	log.Printf("Adding user info: %s", event)
	var userInfo UserSession
	if err := json.Unmarshal(event.Payload, &userInfo); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}
	c.username = userInfo.Username
	c.userID = userInfo.UserID
	c.email = userInfo.Email
	c.joined = userInfo.Joined

	if _, ok := LoggedInList[c.username]; ok {
		log.Println("User in LoggedInList: ", LoggedInList)
	} else if !ok {
		log.Println("Adding user to LoggedInList: ", c.username)
		LoggedInList[c.username] = true
	}

	log.Println("User:", c.username, "added to LoggedInList: ", LoggedInList)

	data, err := json.Marshal(LoggedInList)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message error: %s", err)
	}
	outgoingEvent := Event{
		Payload: json.RawMessage(data),
		Type:    UsersList,
	}

	for c := range c.manager.clients {
		c.egress <- outgoingEvent
	}

	return nil
}

func getChatHistory(event Event, c *Client) error {

	var chtMsg ChatMessage
	if err := json.Unmarshal(event.Payload, &chtMsg); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}
	log.Println("History Request: ", chtMsg)

	rows, err := database.ForumDB.Query("SELECT id, from_user, to_user, is_read, txt, created_at FROM message WHERE (from_user = ? AND to_user = ?) OR (from_user = ? AND to_user = ?) ORDER BY created_at ASC", chtMsg.FromUser, chtMsg.ToUser, chtMsg.ToUser, chtMsg.FromUser)
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

	for c := range c.manager.clients {
		if c.username == chtMsg.FromUser {
			c.egress <- outgoingEvent
			log.Println("History sent to: ", c.username)
		}
	}
	return nil
}

func (m *Manager) RegisterEventHandlers() {
	m.eventHandlers[EventReceiveMessage] = sendMessage
	m.eventHandlers[UserConnect] = addUserInfo
	m.eventHandlers[GetChatHistory] = getChatHistory
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
	log.Println("Client:", client.connection.RemoteAddr(), "added to manager.")
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok { //Checko if client exists in manager
		client.connection.Close()
		delete(m.clients, client)
		log.Println("Client:", client.connection.RemoteAddr(), "removed from manager.")
	}
}

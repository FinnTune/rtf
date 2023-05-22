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
	var broadCastMessage SendMessageEvent
	broadCastMessage.Sent = time.Now()
	broadCastMessage.Message = chatEvent.Message
	broadCastMessage.From = chatEvent.From

	data, err := json.Marshal(broadCastMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message error: %s", err)
	}
	outgoingEvent := Event{
		Payload: data,
		Type:    EventSendMessage,
	}

	for c := range c.manager.clients {
		c.egress <- outgoingEvent
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

// Function to get the conversation history between two users
func getChatHistory(event Event, c *Client) error {
	// Assume that Message is a struct that represents a message in your chat application
	var messages []Message

	// Get the user from the event
	var user string
	if err := json.Unmarshal(event.Payload, &user); err != nil {
		return fmt.Errorf("event unmarshalling error: %s", err)
	}

	// Query the database for the chat history
	rows, err := database.ForumDB.Query(`SELECT * FROM message WHERE from_user = ? OR to_user = ? ORDER BY created_at DESC`, user, user)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var message Message
		err = rows.Scan(&message.ID, &message.From, &message.To, &message.Read, &message.Text, &message.CreatedAt)
		if err != nil {
			return err
		}
		messages = append(messages, message)
	}

	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message error: %s", err)
	}
	outgoingEvent := Event{
		Payload: json.RawMessage(data),
		Type:    "chat_history",
	}

	c.egress <- outgoingEvent

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

// func (m *Manager) removeClient(client *Client) {
// 	m.Lock()
// 	defer m.Unlock()

// 	if _, ok := m.clients[client]; ok { //Checko if client exists in manager
// 		client.connection.Close()
// 		delete(m.clients, client)
// 		log.Println("Client:", client.connection.RemoteAddr(), "removed from manager.")
// 	}
// }

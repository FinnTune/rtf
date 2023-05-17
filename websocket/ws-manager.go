package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

	return nil
}

func (m *Manager) RegisterEventHandlers() {
	m.eventHandlers[EventReceiveMessage] = sendMessage
	m.eventHandlers[UserConnect] = addUserInfo
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

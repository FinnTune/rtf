package websocket

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

type Manager struct {
	clients Clientlist
	sync.RWMutex
	eventHandlers map[string]EventHandler
}

// Factory function for manager
func newManager() *Manager {
	log.Println("Manager created.")
	m := &Manager{
		clients:       make(Clientlist),
		eventHandlers: make(map[string]EventHandler),
	}

	//Register event handlers
	m.RegisterEventHandlers()

	return m
}

func (m *Manager) RegisterEventHandlers() {
	m.eventHandlers[EventSendMessage] = sendMessage
}

func sendMessage(event Event, c *Client) error {
	fmt.Println("Message sent: ", event.Payload)
	log.Printf("Event/message sent: %s", event)
	return nil
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

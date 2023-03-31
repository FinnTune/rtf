package websocket

import (
	"log"
	"sync"
)

type Manager struct {
	clients Clientlist
	sync.RWMutex
}

// Factory function for manager
func newManager() *Manager {
	log.Println("Manager factory function called.")
	return &Manager{
		clients: make(Clientlist),
	}
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true //Add client to manager
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok { //Checko if client exists in manager
		client.connection.Close()
		delete(m.clients, client)
	}
}
package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

// Serve websocket, upgrade incoming requests, and begin client routines for reading and writing messages
func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, m)

	m.addClient(client)

	//Start client routines for reading and writing messages
	go client.readMessages()
	go client.writeMesssage()
}

func StartWebSocket(w http.ResponseWriter, r *http.Request) {
	// Start websocket and manager
	log.Println("Websocket started.")
	manager := newManager()

	manager.ServeWS(w, r)
}

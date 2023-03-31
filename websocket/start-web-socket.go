package websocket

import (
	"fmt"
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

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, m)

	m.addClient(client)
	fmt.Println("Client:", client.connection.RemoteAddr(), "added to manager.")
	//Start client routines for reading and writing messages
	go client.readMessages()
}

func StartWebSocket(w http.ResponseWriter, r *http.Request) {

	// Start websocket and manager
	log.Println("Websocket started.")
	manager := newManager()

	manager.ServeWS(w, r)
}

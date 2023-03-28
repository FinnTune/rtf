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
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type Client struct {
	// id     string
	connection *websocket.Conn
	send       chan []byte
}

type Manager struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func newManager() *Manager {
	return &Manager{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("New client connected.")

	client := &Client{connection: conn, send: make(chan []byte)}
	m.register <- client
	// go client.read()
	// go client.write()

	conn.Close()
}

func StartWebSocket(w http.ResponseWriter, r *http.Request) {

	// Start websocket and manager
	log.Println("Websocket and Manager Started.")
	manager := newManager()

	manager.ServeWS(w, r)
}

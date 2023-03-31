package websocket

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type Clientlist map[*Client]bool

type Client struct {
	// id     string
	connection *websocket.Conn
	manager    *Manager
}

// Factory function for client
func newClient(conn *websocket.Conn, manager *Manager) *Client {
	log.Println("New client connected.")
	return &Client{
		connection: conn,
		manager:    manager,
	}
}

func (c *Client) readMessages() {
	defer func() {
		//clean up - close connection and remove client from manager
		c.connection.Close()
		c.manager.removeClient(c)
	}()
	for {
		msgType, msg, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err)
				c.manager.removeClient(c)
			}
			break
		}
		fmt.Println("Client:", c.connection.RemoteAddr())
		fmt.Println(string(msg))
		fmt.Println("Messagetype: ", msgType)
	}
}

// func (c *Client) writeMesssage() {

// }

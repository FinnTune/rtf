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
	egress     chan []byte
}

// Factory function for client
func newClient(conn *websocket.Conn, manager *Manager) *Client {
	log.Println("New client connected.")
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan []byte, 256),
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
				c.connection.Close()
				c.manager.removeClient(c)
			}
			break
		}

		//Hack to make sure egress is working
		for wsclients := range c.manager.clients {
			wsclients.egress <- msg
		}

		fmt.Println("Client:", c.connection.RemoteAddr())
		fmt.Println(string(msg))
		fmt.Println("Messagetype: ", msgType)
	}
}

func (c *Client) writeMesssage() {
	defer func() {
		c.connection.Close()
		c.manager.removeClient(c)
	}()
	for {
		select {
		case msg, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Printf("Error when writing error message: %s", err)
				}
				log.Printf("Error when writing message to channel: %s", msg)
				return
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println(err)
			}
			log.Println("Message sent to client:", c.connection.RemoteAddr())
		}
	}
}

package websocket

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Clientlist map[*Client]bool

type Client struct {
	// id     string
	connection *websocket.Conn
	manager    *Manager
	//egress is used to avoid concurrent writes to websocket connection
	egress chan Event
}

// Factory function for client
func newClient(conn *websocket.Conn, manager *Manager) *Client {
	log.Println("New client connected.")
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event),
	}
}

func (c *Client) readMessages() {
	log.Println("Client IP and client port num.: ", c.connection.RemoteAddr())
	defer func() {
		//clean up - close connection and remove client from manager
		c.connection.Close()
		c.manager.removeClient(c)
	}()
	for {
		_, msg, err := c.connection.ReadMessage()
		log.Println("Client begin for loop: ", c.connection.RemoteAddr())
		if err != nil {
			log.Println("Client Made an Error: ", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client ReadMessage() error: %s", err)
				c.connection.Close()
				c.manager.removeClient(c)
				//Print address of client connection
				log.Println("Client Inside Error: ", c.connection.RemoteAddr())
			}
			//Break scope and html for submission note.
			//Problem with page refresh upon form submission in html which causes the the connection to close and websocket to resart.
			//Client is closed then the connection ReadMessage function is called for the non-existent client connection and causes panic.
			//If break inside previous if statement, this break will not be executed upon conneciton close when restarting websocket, unless IsUnexpectedCloseError returns true.
			//This will cause the client's connection ReadMessage function to be called without a connection being present.
			break //break out of for loop and triggers the defer cleanup.
		}

		//Hack to make sure egress is working
		// for wsclients := range c.manager.clients {
		// 	wsclients.egress <- msg
		// }

		// fmt.Println("Client: ", c.connection.RemoteAddr())
		// fmt.Println(string(msg))
		// fmt.Println("Messagetype: ", msgType)

		//Replaced above test with the following
		//Unmarshal message into Event struct instance called request
		var request Event

		if err := json.Unmarshal(msg, &request); err != nil {
			log.Printf("Error when unmarshalling msg: %s", err)
			//Maybe a bit harsh to break after one incorret message
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Printf("Error when routing event: %s", err)
			break
		}

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
					log.Printf("Error when writing 'close' message to client: %s", err)
				}
				log.Printf("Error when receiving message from channel 'egress': %s", msg)
				return //break out of for loop/select and triggers the defer cleanup.
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error when marshalling msg: %s", err)
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("Error when writing msg payload to client: %s", err)
			}
			log.Println("Message sent to client:", c.connection.RemoteAddr())
		}
	}
}

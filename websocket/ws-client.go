package websocket

import (
	"encoding/json"
	"log"
	"time"

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

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

// Factory function for client
func newClient(conn *websocket.Conn, manager *Manager) *Client {
	log.Println("New client connected.")
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event),
	}
}

func (c *Client) pongHandler(string) error {
	log.Println("Pong received, handler called, timer reset.")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}

func (c *Client) readMessages() {
	log.Println("Client IP and client port num.: ", c.connection.RemoteAddr())
	defer func() {
		//clean up - close connection and remove client from manager
		c.connection.Close()
		c.manager.removeClient(c)
	}()

	//Set read deadline
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Client SetReadDeadline() error: %s", err)
		return
	}

	//Set pong handler
	c.connection.SetPongHandler(c.pongHandler)

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

	ticker := time.NewTicker(pingInterval)

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

		case <-ticker.C:
			log.Printf("Ping sent to client: %s", c.connection.RemoteAddr())
			if err := c.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error when writing 'ping' message to client: %s", err)
				return
			}

		}
	}
}

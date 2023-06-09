package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type ClientsMapList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	sessionID  string
	//egress is used to avoid concurrent writes to websocket connection
	egress   chan Event
	loggedIn bool
	username string
	userID   int
	email    string
	joined   string
	cookie   *http.Cookie
	// type UserSession struct {
	// 	Username string `json:"username"`
	// 	UserID   int    `json:"id"`
	// 	Email    string `json:"email"`
	// 	Joined   string `json:"joined"`
	// 	Cookie   *http.Cookie
	// }
}

// Initializing variables for ping/pong heartbeat.
// Ping interval must be less than pong wait becuase pong wait is the time the server waits for a pong response.
var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

// Factory function for client
func newClient(conn *websocket.Conn, manager *Manager, session_id string) *Client {
	log.Println("New client connected.")
	return &Client{
		connection: conn,
		manager:    manager,
		sessionID:  session_id,
		egress:     make(chan Event),
		loggedIn:   false,
	}
}

// Function to reset timer after pong is received.
func (c *Client) pongHandler(string) error {
	if c.connection == nil {
		log.Println("Client connection is nil. No pong received.")
		return c.connection.SetReadDeadline(time.Now().Add(pongWait))
	}
	// log.Println("Pong received, handler called, timer reset.")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}

func (c *Client) readMessages() {
	log.Println("Client IP and client port num.: ", c.connection.RemoteAddr())
	defer func() {
		//connection clean up - close connection and remove client from manager
		if c.connection != nil {
			c.connection.Close()
			delete(LoggedInList, c.username)
			c.connection = nil
		}
	}()

	//Set read deadline for pong wait.
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Client SetReadDeadline() error: %s", err)
		return
	}

	//Set limit for message size.
	c.connection.SetReadLimit(512)

	//Set pong handler function for connection
	c.connection.SetPongHandler(c.pongHandler)

	//Go routine for server to read incoming messages from client.
	for {
		_, msg, err := c.connection.ReadMessage()
		log.Println("Client read message from: ", c.connection.RemoteAddr())
		if err != nil {
			delete(LoggedInList, c.username)
			log.Println("Client Made an Error: ", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client ReadMessage() error: %s", err)
				if c.connection != nil {
					c.connection.Close()
					c.connection = nil
				}
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
		// for wsMClientsMapList := range c.manager.MClientsMapList {
		// 	wsMClientsMapList.egress <- msg
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
		if c.connection != nil {
			c.connection.Close()
			delete(LoggedInList, c.username)
			c.connection = nil
		}
	}()

	//Declare new ticker channel with pingInterval
	ticker := time.NewTicker(pingInterval)

	//Go routine for server select case action for incoming channels (msg, ticker...???)
	for {
		select {
		case msg, ok := <-c.egress:
			//Check if channel is closed
			if c.connection == nil {
				log.Println("Client connection is nil.")
				return
			}
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Printf("Error when writing 'close' message to client: %s", err)
					delete(LoggedInList, c.username)
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
			log.Println("Message sent to client:", c.connection.RemoteAddr(), "Message:", string(data))

		case <-ticker.C:
			//Check if channel is closed
			if c.connection == nil {
				log.Println("Client connection is nil.")
				return
			}
			// log.Printf("Ping sent to client: %s", c.connection.RemoteAddr())
			if err := c.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error when writing 'ping' message to client: %s", err)
				return
			}

		}
	}
}

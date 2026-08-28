package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"rtForum/utility"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ClientsMapList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	// connMu guards connection. readMessages and writeMesssage are two
	// independent goroutines per client, each with its own check-then-close
	// cleanup path on this field — a connection drop naturally fails both
	// sides' I/O around the same time, so without this they can race: one
	// goroutine's nil check can pass right before the other goroutine nils
	// the field, and the first goroutine then calls a method on nil.
	// HTTP handlers (checkLogin, serveLogin, ServeWS, ...) touch it too.
	connMu    sync.Mutex
	manager   *Manager
	sessionID string
	//egress is used to avoid concurrent writes to websocket connection
	egress   chan Event
	loggedIn bool
	username string
	userID   int
	email    string
	joined   string
	cookie   *http.Cookie
	// lastSeen backs a sliding session expiry: it is refreshed on every
	// authenticated request and checked against utility.SessionDuration so
	// a leaked/replayed session_id can't be used indefinitely regardless of
	// what the browser does with its own copy of the cookie.
	lastSeen time.Time
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
		lastSeen:   time.Now(),
	}
}

// newAuthenticatedClient creates a Client at the moment a password login
// succeeds, binding sessionID to the identity the server just verified
// against the database. This is the only place a Client's identity fields
// should ever be set from a trusted source — the websocket connection
// itself (see addUserInfo/user-connect) must never be allowed to set or
// change them from client-supplied data, or any authenticated session could
// declare itself to be any other user.
func newAuthenticatedClient(manager *Manager, sessionID string, userID int, username, email, joined string) *Client {
	return &Client{
		manager:   manager,
		sessionID: sessionID,
		egress:    make(chan Event),
		loggedIn:  true,
		userID:    userID,
		username:  username,
		email:     email,
		joined:    joined,
		lastSeen:  time.Now(),
	}
}

// touch refreshes the session's sliding expiry.
func (c *Client) touch() {
	c.lastSeen = time.Now()
}

// expired reports whether the session has been idle longer than
// utility.SessionDuration.
func (c *Client) expired() bool {
	return time.Since(c.lastSeen) > utility.SessionDuration
}

// getConnection safely reads the current connection (nil if none/closed).
func (c *Client) getConnection() *websocket.Conn {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.connection
}

// setConnection safely assigns a (re)connected websocket.
func (c *Client) setConnection(conn *websocket.Conn) {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	c.connection = conn
}

// closeConnection closes and clears the connection if one is set. Safe to
// call concurrently and repeatedly — see the connMu doc comment above.
func (c *Client) closeConnection() {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.connection != nil {
		c.connection.Close()
		c.connection = nil
	}
}

// Function to reset timer after pong is received.
func (c *Client) pongHandler(string) error {
	conn := c.getConnection()
	if conn == nil {
		log.Println("Client connection is nil. No pong received.")
		return nil
	}
	// log.Println("Pong received, handler called, timer reset.")
	return conn.SetReadDeadline(time.Now().Add(pongWait))
}

func (c *Client) readMessages() {
	conn := c.getConnection()
	if conn == nil {
		return
	}
	log.Println("Client IP and client port num.: ", conn.RemoteAddr())
	defer func() {
		//connection clean up - close connection and remove client from manager
		c.closeConnection()
		LoggedInList.Remove(c.username)
	}()

	//Set read deadline for pong wait.
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Client SetReadDeadline() error: %s", err)
		return
	}

	//Set limit for message size.
	conn.SetReadLimit(512)

	//Set pong handler function for connection
	conn.SetPongHandler(c.pongHandler)

	//Go routine for server to read incoming messages from client.
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			LoggedInList.Remove(c.username)
			log.Println("Client Made an Error: ", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client ReadMessage() error: %s", err)
				c.closeConnection()
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
		c.closeConnection()
		LoggedInList.Remove(c.username)
	}()

	//Declare new ticker channel with pingInterval
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	//Go routine for server select case action for incoming channels (msg, ticker...???)
	for {
		select {
		case msg, ok := <-c.egress:
			//Check if channel is closed
			conn := c.getConnection()
			if conn == nil {
				log.Println("Client connection is nil.")
				return
			}
			if !ok {
				if err := conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Printf("Error when writing 'close' message to client: %s", err)
					LoggedInList.Remove(c.username)
				}
				log.Printf("Error when receiving message from channel 'egress': %s", msg)
				return //break out of for loop/select and triggers the defer cleanup.
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error when marshalling msg: %s", err)
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("Error when writing msg payload to client: %s", err)
			}
			log.Println("Message sent to client. Message:", string(data))

		case <-ticker.C:
			//Check if channel is closed
			conn := c.getConnection()
			if conn == nil {
				log.Println("Client connection is nil.")
				return
			}
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error when writing 'ping' message to client: %s", err)
				return
			}

		}
	}
}

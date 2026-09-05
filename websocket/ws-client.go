package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"rtForum/utility"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
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
	// limiter bounds how many events (of any type - chat messages, typing
	// indicators, history requests, ...) this connection may dispatch per
	// second. Every REST write endpoint is already behind an IPRateLimiter
	// (see main.go) specifically to guard against automated abuse; the
	// persistent WS connection - which lets one client fire an unbounded
	// number of DB-writing, broadcast-fanning-out events with no request
	// round-trip in between - had no equivalent until this. Owned per-Client
	// rather than keyed by IP/map, since a Client already lives exactly as
	// long as the connection it bounds.
	limiter *rate.Limiter
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

// wsEventRate/wsEventBurst configure every Client's per-second event
// budget. Generous relative to main.go's writeLimiter (10 req/2s) since a
// single WS connection legitimately multiplexes several low-stakes event
// types at once - e.g. the frontend fires a Typing event on every keystroke,
// not just once per debounce window - so this exists to catch genuine
// flooding (thousands of events/sec from a malicious or runaway client)
// rather than to throttle normal chat use.
//
// Vars, not consts, so tests can shrink them (see SetEventRateLimitForTest)
// instead of needing thousands of real events to prove the limit trips.
var (
	wsEventRate  rate.Limit = 20
	wsEventBurst            = 30
)

func newEventLimiter() *rate.Limiter {
	return rate.NewLimiter(wsEventRate, wsEventBurst)
}

// Factory function for client
func newClient(conn *websocket.Conn, manager *Manager, session_id string) *Client {
	slog.Debug("new client struct created")
	return &Client{
		connection: conn,
		manager:    manager,
		sessionID:  session_id,
		egress:     make(chan Event),
		loggedIn:   false,
		lastSeen:   time.Now(),
		limiter:    newEventLimiter(),
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
		limiter:   newEventLimiter(),
	}
}

// touch refreshes the session's sliding expiry.
func (c *Client) touch() {
	c.lastSeen = time.Now()
}

// sendTimeout bounds how long send() blocks on a client's egress channel.
// egress is unbuffered and only writeMesssage's own goroutine ever reads
// it — a connection that drops without a clean close (no read/write error
// ever surfacing, e.g. a bare network drop) leaves that goroutine gone with
// nothing left to drain it, and readMessages/writeMesssage's cleanup paths
// deliberately don't remove such a client from manager.clients (that map
// lookup is also how ServeWS finds and reuses the same Client object across
// a reconnect — see TestServeWS_ExistingClient_ReconnectsSameClient).
// Without a bound here, one such stale client would hang every future
// broadcast that happens to reach it — an unbuffered, un-timed-out send is
// what turned that into a real, repeatedly-observed outage.
//
// A var, not a const, so tests can shrink it (see SetSendTimeoutForTest in
// testhooks.go) rather than actually waiting out 2 real seconds to prove a
// blocked send gives up.
var sendTimeout = 2 * time.Second

// send delivers an event to this client's egress channel, giving up (and
// reporting failure rather than blocking indefinitely) if writeMesssage
// doesn't drain it within sendTimeout. Every send to a Client's egress
// channel — direct reply or broadcast — should go through this rather than
// writing to egress directly.
func (c *Client) send(event Event) bool {
	select {
	case c.egress <- event:
		return true
	case <-time.After(sendTimeout):
		slog.Warn("timed out delivering event, dropping", "event_type", event.Type, "username", c.username)
		return false
	}
}

// broadcastTo delivers event to every client in recipients concurrently —
// each recipient's send() runs in its own goroutine, so one slow/stuck
// recipient's up-to-sendTimeout wait doesn't delay delivery to the others.
// A sequential loop of send() calls would still be correct (each call is
// itself bounded) but a broadcast to N stale recipients would then take up
// to N*sendTimeout instead of a single sendTimeout for the whole batch.
func broadcastTo(recipients []*Client, event Event) {
	var wg sync.WaitGroup
	for _, recipient := range recipients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			c.send(event)
		}(recipient)
	}
	wg.Wait()
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
		slog.Warn("no pong received: client connection is nil")
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
	slog.Info("client read loop starting", "remote_addr", conn.RemoteAddr())
	defer func() {
		//connection clean up - close connection and remove client from manager
		c.closeConnection()
		LoggedInList.Remove(c.username)
	}()

	//Set read deadline for pong wait.
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Error("failed to set read deadline", "error", err)
		return
	}

	// Set limit for message size. Exceeding this doesn't fail the read
	// gracefully — gorilla closes the connection outright — so it has to
	// comfortably cover every legitimate event's full JSON-encoded size,
	// not just the common case: the largest is create-group-chat, whose
	// payload can carry up to maxGroupMembers (50) usernames plus a name,
	// and a chat message can be up to maxChatMessageLength (1000) runes,
	// worst-case 4 bytes each in UTF-8. 8KiB leaves ample headroom above
	// both without meaningfully weakening the size bound.
	conn.SetReadLimit(8192)

	//Set pong handler function for connection
	conn.SetPongHandler(c.pongHandler)

	//Go routine for server to read incoming messages from client.
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			LoggedInList.Remove(c.username)
			// Fires on every disconnect, including an ordinary tab close —
			// Debug rather than Warn since that's the routine case; the
			// IsUnexpectedCloseError branch below re-logs the genuinely
			// abnormal subset at Warn.
			slog.Debug("client read loop ended", "username", c.username, "error", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("unexpected websocket close", "username", c.username, "error", err)
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
			slog.Warn("failed to unmarshal incoming message", "username", c.username, "error", err)
			//Maybe a bit harsh to break after one incorret message
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			slog.Error("error routing event", "username", c.username, "event_type", request.Type, "error", err)
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
				slog.Warn("client connection is nil, stopping writer", "username", c.username)
				return
			}
			if !ok {
				if err := conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					slog.Warn("failed to write close message", "username", c.username, "error", err)
					LoggedInList.Remove(c.username)
				}
				slog.Info("egress channel closed, stopping writer", "username", c.username)
				return //break out of for loop/select and triggers the defer cleanup.
			}

			// Never log msg/data's content here — it can carry private chat
			// message text (SendMessageEvent.Message), so only the event
			// type and size are safe to record.
			data, err := json.Marshal(msg)
			if err != nil {
				slog.Error("failed to marshal outgoing message", "username", c.username, "event_type", msg.Type, "error", err)
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				slog.Error("failed to write message payload to client", "username", c.username, "event_type", msg.Type, "error", err)
			}
			slog.Debug("message sent to client", "username", c.username, "event_type", msg.Type, "bytes", len(data))

		case <-ticker.C:
			//Check if channel is closed
			conn := c.getConnection()
			if conn == nil {
				slog.Warn("client connection is nil, stopping writer", "username", c.username)
				return
			}
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Warn("failed to write ping message", "username", c.username, "error", err)
				return
			}

		}
	}
}

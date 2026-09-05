package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"rtForum/utility"
	"time"

	"golang.org/x/time/rate"
)

// ResetTestState clears websocket manager and session state between tests.
func ResetTestState() {
	manager = newManager(context.Background())
	LoggedInList.Reset()
	LoggedInUsers = make(map[string]*Client)
}

// SetSendTimeoutForTest temporarily shrinks send()'s egress-channel timeout
// so a test can prove a blocked send gives up without actually waiting out
// the real (2s) production timeout. Returns a restore func; callers should
// defer it.
func SetSendTimeoutForTest(d time.Duration) (restore func()) {
	previous := sendTimeout
	sendTimeout = d
	return func() { sendTimeout = previous }
}

// SetEventRateLimitForTest temporarily shrinks the per-client WS event rate
// limit so a test can prove it trips without sending thousands of real
// events. Only affects clients created after this is called - see
// newEventLimiter. Returns a restore func; callers should defer it.
func SetEventRateLimitForTest(r rate.Limit, burst int) (restore func()) {
	prevRate, prevBurst := wsEventRate, wsEventBurst
	wsEventRate, wsEventBurst = r, burst
	return func() { wsEventRate, wsEventBurst = prevRate, prevBurst }
}

// CheckOriginForTest exposes origin validation for external test packages.
func CheckOriginForTest(r *http.Request) bool {
	return checkOrigin(r)
}

// TestClientHandle provides controlled access to a connected test client.
type TestClientHandle struct {
	client *Client
}

// AddTestClient registers an authenticated client in the manager for tests.
func AddTestClient(sessionID, username string, userID int) *TestClientHandle {
	client := &Client{
		manager:   manager,
		sessionID: sessionID,
		username:  username,
		userID:    userID,
		loggedIn:  true,
		egress:    make(chan Event, 4),
		lastSeen:  time.Now(),
		limiter:   newEventLimiter(),
	}
	manager.clients[client] = true
	return &TestClientHandle{client: client}
}

// AddAuthenticatedClient is an alias for AddTestClient.
func AddAuthenticatedClient(sessionID, username string, userID int) {
	AddTestClient(sessionID, username, userID)
}

func (h *TestClientHandle) Username() string { return h.client.username }
func (h *TestClientHandle) UserID() int      { return h.client.userID }

// ExpireForTest backdates the client's lastSeen so client.expired() reports
// true, for testing stale-session cleanup paths without waiting out the
// real SessionDuration.
func (h *TestClientHandle) ExpireForTest() {
	h.client.lastSeen = time.Now().Add(-utility.SessionDuration - time.Minute)
}

// CloseConnectionForTest exercises the connection-close path for tests.
func (h *TestClientHandle) CloseConnectionForTest() { h.client.closeConnection() }

// HasConnectionForTest reports whether the client currently has a
// connection set.
func (h *TestClientHandle) HasConnectionForTest() bool { return h.client.getConnection() != nil }

// ClearConnectionForTest exercises the connection-write path for tests
// without needing a real *websocket.Conn.
func (h *TestClientHandle) ClearConnectionForTest() { h.client.setConnection(nil) }

// SetLoggedInList marks a username as logged in for test setup.
func SetLoggedInList(username string) { LoggedInList.Add(username) }

// IsInLoggedInList reports whether a username is in the online users list.
func IsInLoggedInList(username string) bool { return LoggedInList.Has(username) }

// IsRemovedFromManager reports whether the client was removed from the manager.
func (h *TestClientHandle) IsRemovedFromManager() bool {
	manager.RLock()
	defer manager.RUnlock()
	_, ok := manager.clients[h.client]
	return !ok
}

// NewOtpForTest mints a one-time password against the package's live
// manager — the same instance WebsocketHandler/ServeWS use — for tests that
// exercise the real /ws upgrade path end-to-end rather than calling an
// event handler directly.
func NewOtpForTest() string {
	return manager.otps.newOtp().Key
}

// FindClientBySessionForTest looks up a connected client by session id
// against the package's live manager. Needed for asserting on clients
// ServeWS itself creates or reconnects, which AddTestClient's handle
// doesn't observe.
func FindClientBySessionForTest(sessionID string) *TestClientHandle {
	manager.RLock()
	defer manager.RUnlock()
	for c := range manager.clients {
		if c.sessionID == sessionID {
			return &TestClientHandle{client: c}
		}
	}
	return nil
}

// ClientCountForTest reports how many clients are currently registered with
// the package's live manager.
func ClientCountForTest() int {
	manager.RLock()
	defer manager.RUnlock()
	return len(manager.clients)
}

// SweepExpiredClientsForTest runs one pass of the background expired-client
// eviction synchronously, so a test can assert on its effect without
// waiting out the real clientSweepInterval.
func SweepExpiredClientsForTest() {
	manager.sweepOnce()
}

// WaitEvent waits for an outbound websocket event up to the given timeout.
func (h *TestClientHandle) WaitEvent(timeout time.Duration) (eventType string, payload json.RawMessage, ok bool) {
	select {
	case evt := <-h.client.egress:
		return evt.Type, evt.Payload, true
	case <-time.After(timeout):
		return "", nil, false
	}
}

func dispatchEvent(event Event, client *TestClientHandle) error {
	return manager.routeEvent(event, client.client)
}

// RouteEventForTest routes an event through the manager for tests.
func RouteEventForTest(eventType string, payload json.RawMessage, client *TestClientHandle) error {
	return dispatchEvent(Event{Type: eventType, Payload: payload}, client)
}

// SendMessageForTest invokes the chat message handler for tests.
func SendMessageForTest(payload json.RawMessage, client *TestClientHandle) error {
	return sendMessage(Event{Type: EventReceiveMessage, Payload: payload}, client.client)
}

// AddUserInfoForTest invokes the user-connect handler for tests.
func AddUserInfoForTest(payload json.RawMessage, client *TestClientHandle) error {
	return addUserInfo(Event{Type: UserConnect, Payload: payload}, client.client)
}

// GetChatHistoryForTest invokes the chat history handler for tests.
func GetChatHistoryForTest(payload json.RawMessage, client *TestClientHandle) error {
	return getChatHistory(Event{Type: GetChatHistory, Payload: payload}, client.client)
}

// TypingForTest invokes the typing indicator handler for tests.
func TypingForTest(payload json.RawMessage, client *TestClientHandle) error {
	return typing(Event{Type: Typing, Payload: payload}, client.client)
}

// StopTypingForTest invokes the stop-typing handler for tests.
func StopTypingForTest(payload json.RawMessage, client *TestClientHandle) error {
	return stopTyping(Event{Type: StopTyping, Payload: payload}, client.client)
}

// OpenDirectChatForTest invokes the open-direct-chat handler for tests.
func OpenDirectChatForTest(payload json.RawMessage, client *TestClientHandle) error {
	return openDirectChat(Event{Type: OpenDirectChat, Payload: payload}, client.client)
}

// CreateGroupChatForTest invokes the create-group-chat handler for tests.
func CreateGroupChatForTest(payload json.RawMessage, client *TestClientHandle) error {
	return createGroupChat(Event{Type: CreateGroupChat, Payload: payload}, client.client)
}

// GetConversationsForTest invokes the get-conversations handler for tests.
func GetConversationsForTest(client *TestClientHandle) error {
	return getConversations(Event{Type: GetConversations}, client.client)
}

// MarkReadForTest invokes the mark-read handler for tests.
func MarkReadForTest(payload json.RawMessage, client *TestClientHandle) error {
	return markRead(Event{Type: MarkRead, Payload: payload}, client.client)
}

// TestOtps wraps OTP map lifecycle for external tests.
type TestOtps struct {
	otps   *otpsMap
	cancel context.CancelFunc
}

// NewTestOtps creates an OTP map with the given expiry duration.
func NewTestOtps(expiry time.Duration) *TestOtps {
	ctx, cancel := context.WithCancel(context.Background())
	return &TestOtps{otps: newOtpsMap(ctx, expiry), cancel: cancel}
}

// Close stops background OTP cleanup.
func (o *TestOtps) Close() { o.cancel() }

// NewKey creates a new OTP and returns its key.
func (o *TestOtps) NewKey() string { return o.otps.newOtp().Key }

// Verify validates and consumes an OTP key.
func (o *TestOtps) Verify(key string) bool { return o.otps.verifyOtp(key) }

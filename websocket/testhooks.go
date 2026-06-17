package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// ResetTestState clears websocket manager and session state between tests.
func ResetTestState() {
	manager = newManager(context.Background())
	LoggedInList = make(map[string]bool)
	LoggedInUsers = make(map[string]*Client)
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
	}
	manager.clients[client] = true
	return &TestClientHandle{client: client}
}

// AddAuthenticatedClient is an alias for AddTestClient.
func AddAuthenticatedClient(sessionID, username string, userID int) {
	AddTestClient(sessionID, username, userID)
}

func (h *TestClientHandle) Username() string { return h.client.username }
func (h *TestClientHandle) UserID() int       { return h.client.userID }

// SetLoggedInList marks a username as logged in for test setup.
func SetLoggedInList(username string) { LoggedInList[username] = true }

// IsInLoggedInList reports whether a username is in the online users list.
func IsInLoggedInList(username string) bool { return LoggedInList[username] }

// IsRemovedFromManager reports whether the client was removed from the manager.
func (h *TestClientHandle) IsRemovedFromManager() bool {
	manager.RLock()
	defer manager.RUnlock()
	_, ok := manager.clients[h.client]
	return !ok
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

// TestOtps wraps OTP map lifecycle for external tests.
type TestOtps struct {
	otps   otpsMap
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

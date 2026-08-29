package websocket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"rtForum/websocket"

	gorillaws "github.com/gorilla/websocket"
)

// dialWS attempts the real /ws upgrade handshake against a running test
// server, mirroring what a browser does: an Origin header matching the
// default allowed origin, a query-string otp, and (unless empty) a
// session_id cookie.
func dialWS(t *testing.T, serverURL, otp, sessionID string) (*gorillaws.Conn, *http.Response, error) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws"
	if otp != "" {
		wsURL += "?otp=" + otp
	}
	header := http.Header{}
	header.Set("Origin", "https://localhost:8443")
	if sessionID != "" {
		header.Set("Cookie", "session_id="+sessionID)
	}
	return gorillaws.DefaultDialer.Dial(wsURL, header)
}

func TestServeWS_MissingOTP_Rejected(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestServeWS_InvalidOTP_Rejected(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws?otp=not-a-real-otp")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestServeWS_WrongOrigin_Rejected(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	otp := websocket.NewOtpForTest()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?otp=" + otp
	header := http.Header{}
	header.Set("Origin", "https://evil.example.com")
	header.Set("Cookie", "session_id=whatever")

	_, resp, err := gorillaws.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected the handshake to fail for a disallowed origin")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		status := -1
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, status)
	}
}

func TestServeWS_ValidOTP_MissingCookie_ConnectionClosed(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	otp := websocket.NewOtpForTest()
	conn, resp, err := dialWS(t, server.URL, otp, "")
	if err != nil {
		t.Fatalf("expected the upgrade itself to succeed (cookie is checked after), got error: %v (status %v)", err, resp)
	}
	defer conn.Close()

	// The server closes the connection right after upgrading, since it has
	// no session cookie to attach the connection to. The client should
	// observe the connection being closed rather than it staying open.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, readErr := conn.ReadMessage()
	if readErr == nil {
		t.Fatal("expected the connection to be closed by the server due to the missing session cookie")
	}

	if got := websocket.ClientCountForTest(); got != 0 {
		t.Fatalf("expected no client to be registered without a session cookie, got %d", got)
	}
}

func TestServeWS_NewClient_UpgradesRegistersAndRoutesEvents(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	otp := websocket.NewOtpForTest()
	sessionID := "brand-new-session"
	conn, _, err := dialWS(t, server.URL, otp, sessionID)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// Give ServeWS's goroutines a moment to register the client.
	deadlineCheck := time.Now().Add(2 * time.Second)
	for websocket.ClientCountForTest() == 0 && time.Now().Before(deadlineCheck) {
		time.Sleep(10 * time.Millisecond)
	}

	if got := websocket.ClientCountForTest(); got != 1 {
		t.Fatalf("expected exactly 1 registered client, got %d", got)
	}
	handle := websocket.FindClientBySessionForTest(sessionID)
	if handle == nil {
		t.Fatal("expected to find the new client by session id")
	}
	if !handle.HasConnectionForTest() {
		t.Fatal("expected the new client to have its connection set")
	}

	// End-to-end proof that readMessages and writeMesssage are both
	// functioning together on this connection: send a real user-connect
	// event over the wire and expect the server to broadcast users-online
	// back, exercising the exact concurrent connection-lifecycle code paths
	// fixed earlier this session.
	userConnectEvent := map[string]any{
		"type":    "user-connect",
		"payload": map[string]any{"username": "brandnewuser", "id": 1},
	}
	data, _ := json.Marshal(userConnectEvent)
	if err := conn.WriteMessage(gorillaws.TextMessage, data); err != nil {
		t.Fatalf("failed to send user-connect: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, reply, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("expected a broadcast reply after user-connect, got error: %v", err)
	}
	var replyEvent struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(reply, &replyEvent); err != nil {
		t.Fatalf("failed to decode reply: %v", err)
	}
	if replyEvent.Type != websocket.UsersList {
		t.Fatalf("expected a %q broadcast, got %q", websocket.UsersList, replyEvent.Type)
	}
}

func TestServeWS_ExistingClient_ReconnectsSameClient(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "existing-session"
	websocket.AddAuthenticatedClient(sessionID, "reconnector", 7)
	if got := websocket.ClientCountForTest(); got != 1 {
		t.Fatalf("expected 1 pre-registered client, got %d", got)
	}

	otp := websocket.NewOtpForTest()
	conn, _, err := dialWS(t, server.URL, otp, sessionID)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	deadlineCheck := time.Now().Add(2 * time.Second)
	for {
		handle := websocket.FindClientBySessionForTest(sessionID)
		if handle != nil && handle.HasConnectionForTest() {
			break
		}
		if time.Now().After(deadlineCheck) {
			t.Fatal("timed out waiting for the existing client's connection to be set")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Reconnecting an existing session must reuse the same client, not
	// register a second one.
	if got := websocket.ClientCountForTest(); got != 1 {
		t.Fatalf("expected reconnect to keep the client count at 1, got %d", got)
	}
}

func TestServeWS_ExpiredExistingClient_DiscardedAndReplaced(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "stale-session"
	websocket.AddAuthenticatedClient(sessionID, "staleuser", 9)
	handle := websocket.FindClientBySessionForTest(sessionID)
	if handle == nil {
		t.Fatal("expected the pre-registered stale client to be findable")
	}
	handle.ExpireForTest()

	otp := websocket.NewOtpForTest()
	conn, _, err := dialWS(t, server.URL, otp, sessionID)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// A stale client is already registered before the dial (count starts at
	// 1), so — unlike the other tests here — waiting on the count alone
	// can't distinguish "still the old stale entry" from "discarded and
	// replaced". ServeWS also finishes registering the new client
	// asynchronously after the client-side Dial() call already returns (it
	// returns as soon as the HTTP 101 upgrade completes, before the
	// server's goroutine gets to the cookie check / stale-client deletion /
	// new-client registration that follow). Poll on the actual condition:
	// the new client, with its connection set.
	deadlineCheck := time.Now().Add(2 * time.Second)
	var newHandle *websocket.TestClientHandle
	for time.Now().Before(deadlineCheck) {
		newHandle = websocket.FindClientBySessionForTest(sessionID)
		if newHandle != nil && newHandle.HasConnectionForTest() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if newHandle == nil || !newHandle.HasConnectionForTest() {
		t.Fatal("timed out waiting for a new, connected client to be registered under the same session id")
	}
	if got := websocket.ClientCountForTest(); got != 1 {
		t.Fatalf("expected the stale client to be discarded and replaced by exactly 1 new client, got %d", got)
	}
}

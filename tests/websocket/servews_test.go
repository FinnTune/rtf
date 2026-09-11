package websocket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"rtForum/tests/testutil"
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

	// A real session cookie, so this actually exercises otp verification
	// itself rather than failing earlier on the missing-cookie check.
	req, err := http.NewRequest(http.MethodGet, server.URL+"/ws?otp=not-a-real-otp", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "some-session"})

	resp, err := http.DefaultClient.Do(req)
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

	sessionID := "whatever"
	otp := websocket.NewOtpForTest(sessionID)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?otp=" + otp
	header := http.Header{}
	header.Set("Origin", "https://evil.example.com")
	header.Set("Cookie", "session_id="+sessionID)

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

func TestServeWS_ValidOTP_MissingCookie_Rejected(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	// The session cookie is read before the otp is verified against it (see
	// ServeWS), so a missing cookie is rejected outright — the upgrade
	// itself never happens, regardless of how the otp was minted.
	otp := websocket.NewOtpForTest("some-session")
	conn, resp, err := dialWS(t, server.URL, otp, "")
	if err == nil {
		conn.Close()
		t.Fatal("expected the handshake to fail without a session cookie")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		status := -1
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, status)
	}

	if got := websocket.ClientCountForTest(); got != 0 {
		t.Fatalf("expected no client to be registered without a session cookie, got %d", got)
	}
}

// TestServeWS_OTPMintedForDifferentSession_Rejected is the end-to-end
// regression test for the otp/session binding fix: an otp is only ever
// minted by checkLogin/serveLogin for the caller's own, already-verified
// session (see those handlers). Before this fix, ServeWS accepted any
// live, unused otp paired with *any* session_id cookie — so anyone who
// could log in once could mint an otp for their own session and redeem it
// here against an arbitrary, made-up session_id, hitting the
// no-existing-client branch below and registering a fresh, loggedIn Client
// with userID 0 and an empty username. Confirms the otp/session binding is
// enforced by ServeWS itself, not just by the otpsMap in isolation (see
// otp_test.go's TestVerifyOtp_WrongSessionRejected for that narrower case).
func TestServeWS_OTPMintedForDifferentSession_Rejected(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	otp := websocket.NewOtpForTest("real-session")
	conn, resp, err := dialWS(t, server.URL, otp, "attacker-made-up-session")
	if err == nil {
		conn.Close()
		t.Fatal("expected the handshake to fail when the otp was minted for a different session")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		status := -1
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, status)
	}

	if got := websocket.ClientCountForTest(); got != 0 {
		t.Fatalf("expected no phantom client to be registered, got %d", got)
	}
}

func TestServeWS_NewClient_UpgradesRegistersAndRoutesEvents(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "brand-new-session"
	otp := websocket.NewOtpForTest(sessionID)
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

	otp := websocket.NewOtpForTest(sessionID)
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

	otp := websocket.NewOtpForTest(sessionID)
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

// TestCheckLogin_DoesNotCloseExistingConnection covers the multi-tab
// scenario: browser tabs on the same domain share one session_id cookie, so
// a second tab's checkLogin call (fired on every mount by both AuthContext
// and WebSocketContext, before it opens its own socket) looks up the exact
// same Client as an already-connected first tab. checkLogin previously
// force-closed that connection unconditionally, which meant merely opening
// (or even just loading, without ever completing its own connection) a
// second tab would kill an unrelated, healthy first tab's live socket.
func TestCheckLogin_DoesNotCloseExistingConnection(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "shared-session"
	// checkLogin looks up role/banned status by user id — user 42
	// ("actual_user") is real seed data.
	websocket.AddAuthenticatedClient(sessionID, "actual_user", 42)

	otp := websocket.NewOtpForTest(sessionID)
	conn, _, err := dialWS(t, server.URL, otp, sessionID)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	var handle *websocket.TestClientHandle
	deadlineCheck := time.Now().Add(2 * time.Second)
	for {
		handle = websocket.FindClientBySessionForTest(sessionID)
		if handle != nil && handle.HasConnectionForTest() {
			break
		}
		if time.Now().After(deadlineCheck) {
			t.Fatal("timed out waiting for the client's connection to be set")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Simulate a second tab (same shared cookie) polling status/minting its
	// own OTP before it has opened any socket of its own.
	req := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	rr := httptest.NewRecorder()
	websocket.CheckLoginHandler(rr, req)

	var resp websocket.UserLoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode checkLogin response: %v", err)
	}
	if !resp.LoggedIn || resp.OTP == "" {
		t.Fatalf("expected checkLogin to report logged in with a fresh OTP, got %+v", resp)
	}

	if !handle.HasConnectionForTest() {
		t.Fatal("expected the first tab's live connection to survive a second tab's checkLogin call")
	}

	// Confirm the connection is genuinely still usable end-to-end, not just
	// non-nil on the server struct.
	userConnectEvent := map[string]any{
		"type":    "user-connect",
		"payload": map[string]any{"username": "actual_user", "id": 42},
	}
	data, _ := json.Marshal(userConnectEvent)
	if err := conn.WriteMessage(gorillaws.TextMessage, data); err != nil {
		t.Fatalf("expected the surviving connection to still accept writes: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("expected the surviving connection to still receive a reply: %v", err)
	}
}

// TestServeWS_ExistingClient_ClosesPreviousConnectionBeforeReplacing covers
// the other half of the same fix: when a genuinely new connection *does*
// attach for a session that already has a live one (a real second tab, not
// just a checkLogin poll), the old connection must be closed rather than
// silently orphaned — otherwise its still-running writeMesssage goroutine
// (which re-fetches the current connection every loop iteration rather than
// caching it) would start writing to the new connection alongside the new
// goroutine ServeWS starts for it.
func TestServeWS_ExistingClient_ClosesPreviousConnectionBeforeReplacing(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "two-tabs-session"
	firstOTP := websocket.NewOtpForTest(sessionID)
	firstConn, _, err := dialWS(t, server.URL, firstOTP, sessionID)
	if err != nil {
		t.Fatalf("first dial failed: %v", err)
	}
	defer firstConn.Close()

	deadlineCheck := time.Now().Add(2 * time.Second)
	for {
		handle := websocket.FindClientBySessionForTest(sessionID)
		if handle != nil && handle.HasConnectionForTest() {
			break
		}
		if time.Now().After(deadlineCheck) {
			t.Fatal("timed out waiting for the first connection to be set")
		}
		time.Sleep(10 * time.Millisecond)
	}

	secondOTP := websocket.NewOtpForTest(sessionID)
	secondConn, _, err := dialWS(t, server.URL, secondOTP, sessionID)
	if err != nil {
		t.Fatalf("second dial failed: %v", err)
	}
	defer secondConn.Close()

	// The first tab's underlying socket must actually be closed by the
	// server, not left dangling. closeConnection() runs strictly before
	// setConnection(secondConn) in ServeWS's existing-client branch, but
	// that doesn't mean the peer observes the close only after the rest of
	// that branch has run too — the two happen on different ends of the
	// connection, so this read returning is not itself proof the second
	// connection is registered yet; poll for that separately below.
	firstConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := firstConn.ReadMessage(); err == nil {
		t.Fatal("expected the first connection to be closed once a second connection replaced it")
	}

	if got := websocket.ClientCountForTest(); got != 1 {
		t.Fatalf("expected still exactly 1 client (same session, reused), got %d", got)
	}

	// Confirm the second connection actually gets registered — send a real
	// frame over it and expect a reply. The write itself succeeds as soon
	// as the TCP-level upgrade completes regardless of server-side
	// progress, but the reply can only arrive once ServeWS's
	// existing-client branch has finished setConnection/touch and started
	// this connection's own readMessages/writeMesssage goroutines, so the
	// blocking read below is what actually waits out that async handoff.
	userConnectEvent := map[string]any{
		"type":    "user-connect",
		"payload": map[string]any{"username": "tabuser2", "id": 12},
	}
	data, _ := json.Marshal(userConnectEvent)
	if err := secondConn.WriteMessage(gorillaws.TextMessage, data); err != nil {
		t.Fatalf("failed to write on the second connection: %v", err)
	}
	secondConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := secondConn.ReadMessage(); err != nil {
		t.Fatalf("expected the second connection to be live end-to-end: %v", err)
	}

	handle := websocket.FindClientBySessionForTest(sessionID)
	if handle == nil || !handle.HasConnectionForTest() {
		t.Fatal("expected the session's client to have the second connection set")
	}
}

// TestServeWS_ConcurrentNewSessionDials_RegisterExactlyOneClient covers a
// gap the sequential-replacement tests above can't reach: two genuinely
// concurrent ServeWS calls for the same, brand-new session_id (e.g. two
// browser tabs opening within the same instant — nothing ties an OTP to a
// single tab, so both independently minting and using their own valid OTP
// is normal). Before ServeWS held m.Lock() across the whole lookup-or-create
// decision, both goroutines could concurrently find "no existing client"
// and each create and register a separate *Client for the same session_id,
// leaving manager.clients with two entries other handlers could
// inconsistently pick between.
func TestServeWS_ConcurrentNewSessionDials_RegisterExactlyOneClient(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "concurrent-new-session"
	// More than 2 concurrent dials, released together via a start barrier
	// (rather than just launched in a loop), to maximize how many of them
	// actually overlap inside ServeWS instead of happening to serialize —
	// this is what makes the test a reliable regression guard rather than a
	// low-probability reproduction (an earlier 2-dial version of this test
	// only caught the pre-fix bug roughly 1 run in 15).
	const n = 20
	otps := make([]string, n)
	for i := range otps {
		otps[i] = websocket.NewOtpForTest(sessionID)
	}

	var start sync.WaitGroup
	start.Add(1)
	var wg sync.WaitGroup
	conns := make([]*gorillaws.Conn, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start.Wait()
			conns[i], _, errs[i] = dialWS(t, server.URL, otps[i], sessionID)
		}(i)
	}
	start.Done() // release all n goroutines at once
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("dial %d failed: %v", i, err)
		}
		defer conns[i].Close()
	}

	deadlineCheck := time.Now().Add(2 * time.Second)
	for websocket.ClientCountForTest() == 0 && time.Now().Before(deadlineCheck) {
		time.Sleep(10 * time.Millisecond)
	}
	// A grace window for further, possibly-buggy registrations to also
	// land — without this, a fast-but-wrong run (only one goroutine has
	// registered so far) would look identical to a correct one.
	time.Sleep(200 * time.Millisecond)

	if got := websocket.ClientCountForTest(); got != 1 {
		t.Fatalf("expected exactly 1 registered client for one session_id even under %d concurrent dials, got %d", n, got)
	}
}

// TestServeWS_Reconnect_KeepsUserOnlineInLoggedInList covers the same bug
// class as TestServeWS_ExistingClient_ClosesPreviousConnectionBeforeReplacing,
// just on LoggedInList instead of Client.connection: ServeWS's existing-
// client branch re-adds the user to LoggedInList before closing the old
// connection and starting the new one, but the old connection's own
// readMessages/writeMesssage goroutines unblock (with a read/write error)
// once that close happens and, without a currentness guard, would
// unconditionally remove the user again — racing ahead of (and undoing) the
// re-add, since the old goroutine's local error handling needs no network
// round trip while the new connection's own eventual re-add (a real
// user-connect from the frontend) does.
func TestServeWS_Reconnect_KeepsUserOnlineInLoggedInList(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "reconnect-session"
	username := "reconnectuser"
	websocket.AddAuthenticatedClient(sessionID, username, 55)

	firstOTP := websocket.NewOtpForTest(sessionID)
	firstConn, _, err := dialWS(t, server.URL, firstOTP, sessionID)
	if err != nil {
		t.Fatalf("first dial failed: %v", err)
	}
	defer firstConn.Close()

	deadlineCheck := time.Now().Add(2 * time.Second)
	for {
		handle := websocket.FindClientBySessionForTest(sessionID)
		if handle != nil && handle.HasConnectionForTest() {
			break
		}
		if time.Now().After(deadlineCheck) {
			t.Fatal("timed out waiting for the first connection to be set")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Mirrors the frontend sending user-connect right after the first
	// socket opens — establishes the "currently online" state a real
	// reconnect race actually starts from.
	websocket.SetLoggedInList(username)

	secondOTP := websocket.NewOtpForTest(sessionID)
	secondConn, _, err := dialWS(t, server.URL, secondOTP, sessionID)
	if err != nil {
		t.Fatalf("second dial failed: %v", err)
	}
	defer secondConn.Close()

	// Wait for the first connection to actually be closed by the server —
	// the same synchronization point TestServeWS_ExistingClient_Closes...
	// uses — proving ServeWS's existing-client branch (Remove+Add, then
	// close/replace) has fully run.
	firstConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := firstConn.ReadMessage(); err == nil {
		t.Fatal("expected the first connection to be closed once the second replaced it")
	}

	// Give the first connection's own read/write goroutines a moment to run
	// their deferred cleanup — near-instantaneous in practice, since
	// detecting an already-closed connection needs no further I/O. This is
	// exactly the window in which the bug being tested would incorrectly
	// remove the user.
	time.Sleep(200 * time.Millisecond)

	if !websocket.IsInLoggedInList(username) {
		t.Fatal("expected the user to remain in LoggedInList after a reconnect settles, not be removed by the stale connection's own cleanup")
	}
}

// TestClient_GenuineDisconnect_RemovesFromLoggedInList confirms the fix
// above doesn't over-correct into never removing anyone: when a connection
// closes with nothing replacing it, the user must still come off
// LoggedInList.
func TestClient_GenuineDisconnect_RemovesFromLoggedInList(t *testing.T) {
	websocket.ResetTestState()
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "genuine-disconnect-session"
	username := "solouser"
	// AddAuthenticatedClient (rather than a bare new connection) ensures the
	// dial below goes through ServeWS's existing-client branch, so c.username
	// is actually set — a brand-new connection has no identity until a real
	// user-connect event runs, which this test never sends.
	websocket.AddAuthenticatedClient(sessionID, username, 77)

	otp := websocket.NewOtpForTest(sessionID)
	conn, _, err := dialWS(t, server.URL, otp, sessionID)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	deadlineCheck := time.Now().Add(2 * time.Second)
	for {
		handle := websocket.FindClientBySessionForTest(sessionID)
		if handle != nil && handle.HasConnectionForTest() {
			break
		}
		if time.Now().After(deadlineCheck) {
			t.Fatal("timed out waiting for the connection to be set")
		}
		time.Sleep(10 * time.Millisecond)
	}
	websocket.SetLoggedInList(username)

	conn.Close() // genuine client-side disconnect — nothing replaces it

	deadlineCheck = time.Now().Add(2 * time.Second)
	for websocket.IsInLoggedInList(username) {
		if time.Now().After(deadlineCheck) {
			t.Fatal("timed out waiting for a genuine disconnect to remove the user from LoggedInList")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestServeWS_AcceptsChatMessageFrameLargerThanOldReadLimit(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	sessionID := "large-frame-session"
	websocket.AddAuthenticatedClient(sessionID, "admin", 1)

	otp := websocket.NewOtpForTest(sessionID)
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
			t.Fatal("timed out waiting for the client's connection to be set")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// ~700 bytes of message text: comfortably over the old 512-byte frame
	// read limit — which used to kill the whole connection outright (see
	// ws-client.go's SetReadLimit) instead of letting sendMessage reject an
	// over-length message gracefully — but safely under the current one.
	largeMessage := strings.Repeat("a", 700)
	event := map[string]any{
		"type":    "new-message",
		"payload": map[string]any{"conversation_id": 1, "message": largeMessage},
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}
	if err := conn.WriteMessage(gorillaws.TextMessage, data); err != nil {
		t.Fatalf("failed to send large message frame: %v", err)
	}

	// A message-ack reply proves the connection survived the frame and the
	// message was processed end-to-end (admin/user 1 is a member of the
	// seed data's conversation 1) — not just that the socket happened to
	// stay open.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, reply, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("expected the connection to survive the large frame and reply, got error: %v", err)
	}
	var replyEvent struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(reply, &replyEvent); err != nil {
		t.Fatalf("failed to decode reply: %v", err)
	}
	if replyEvent.Type != "message-ack" {
		t.Fatalf("expected message-ack reply, got %q", replyEvent.Type)
	}
}

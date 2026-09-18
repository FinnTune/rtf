package websocket_test

import (
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"strings"
	"testing"
	"time"

	"rtForum/websocket"

	gorillaws "github.com/gorilla/websocket"
)

// TestRouteEventSafely_RecoversFromHandlerPanic is the regression test for
// the panic-recovery fix: readMessages routes every incoming event through
// routeEventSafely rather than calling manager.routeEvent directly, so a
// panic anywhere in a handler's call chain — a nil dereference, an
// out-of-range index, any mistake a handler introduces — is caught here
// and converted into an ordinary error, instead of propagating out of this
// goroutine and crashing the entire process. Go's net/http only
// auto-recovers panics in the goroutine directly serving an HTTP request;
// readMessages runs in its own goroutine spawned from ServeWS, outside
// that protection.
func TestRouteEventSafely_RecoversFromHandlerPanic(t *testing.T) {
	websocket.ResetTestState()
	client := websocket.AddTestClient("session-panic", "alice", 2)

	restore := websocket.SetEventHandlerForTest("boom", func(event websocket.Event, c *websocket.Client) error {
		panic("simulated handler bug")
	})
	defer restore()

	err := websocket.RouteEventSafelyForTest("boom", nil, client)
	if err == nil {
		t.Fatal("expected routeEventSafely to convert the panic into an error, got nil")
	}
	if !strings.Contains(err.Error(), "panic while routing event") {
		t.Fatalf("expected a panic-wrapping error message, got: %v", err)
	}
}

// TestServeWS_HandlerPanic_ClosesOnlyThatConnection is the end-to-end
// version of the same fix, through a real WS connection and the real
// readMessages goroutine (not routeEventSafely called directly). It
// deliberately doesn't rely on the panicked connection's own read
// returning an error as proof of anything: an unrecovered panic in that
// goroutine would eventually tear down the whole process, including this
// test binary itself, and a plain client-side read error can't tell "the
// connection was closed gracefully" apart from "the whole process is in
// the middle of dying." The unambiguous proof the server (not just this
// connection) is still alive is a brand new connection completing a full,
// real round trip afterward — using get-conversations specifically
// (replies directly to the sender only) rather than user-connect, which
// would broadcast to every registered client including the panicked one.
//
// The wait below matters, and has to exceed send()'s sendTimeout (2s,
// unmodified — see why below): an unrecovered panic here doesn't crash the
// process instantly. readMessages' own deferred cleanup still runs during
// the panic's stack unwind (Go runs a goroutine's deferred functions on
// the way to a fatal, unrecovered panic, same as on any other return
// path) — and that cleanup's last step, broadcastUsersList, blocks on this
// very client's own now-abandoned egress channel (its writeMesssage
// goroutine already stopped draining it, since closeConnectionIfCurrent
// closes connDone earlier in that same deferred func) until send() gives
// up. A wait shorter than sendTimeout would let a fast, unrelated process
// exit race ahead of that slower in-flight crash and mask a real
// regression — confirmed by hand while developing this test: at a few
// hundred milliseconds this test passed even against a
// deliberately-reverted, unrecovered version of readMessages.
//
// This deliberately does NOT use SetSendTimeoutForTest to shrink that
// wait: its own restore (a write to the package-level sendTimeout var,
// deferred until this function returns) raced under -race against the
// panicked client's own send() goroutine (a read of that same var) still
// in flight when the deferred restore fired — sendTimeoutForTest's own doc
// comment warns every caller about exactly this. Rather than chase ever
// larger settling-sleep margins to outrun a background goroutine whose
// completion isn't directly observable from the test, this uses the real,
// unmodified 2s timeout and simply waits comfortably longer than it.
func TestServeWS_HandlerPanic_ClosesOnlyThatConnection(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	server := httptest.NewServer(http.HandlerFunc(websocket.WebsocketHandler))
	defer server.Close()

	restore := websocket.SetEventHandlerForTest("boom", func(event websocket.Event, c *websocket.Client) error {
		panic("simulated handler bug")
	})
	defer restore()

	sessionID := "panic-session"
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
			t.Fatal("timed out waiting for the connection to be set")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := conn.WriteMessage(gorillaws.TextMessage, []byte(`{"type":"boom","payload":{}}`)); err != nil {
		t.Fatalf("failed to send boom event: %v", err)
	}

	// See the doc comment above for why this has to comfortably exceed the
	// real, unmodified 2s sendTimeout.
	time.Sleep(3 * time.Second)

	handle := websocket.FindClientBySessionForTest(sessionID)
	if handle == nil {
		t.Fatal("expected the client to still be registered under its session id after a panic-triggered disconnect")
	}
	if handle.HasConnectionForTest() {
		t.Fatal("expected the panicked connection to have been closed, same as any other routing error")
	}

	// The real proof: the server itself is still alive and functioning for
	// every OTHER connection — dial a brand new one and complete a full
	// round trip. If the process had actually crashed, this test would
	// never have gotten this far to try.
	otherSessionID := "after-panic-session"
	otherOtp := websocket.NewOtpForTest(otherSessionID)
	otherConn, _, err := dialWS(t, server.URL, otherOtp, otherSessionID)
	if err != nil {
		t.Fatalf("expected the server to still accept new connections after the panic, dial failed: %v", err)
	}
	defer otherConn.Close()

	if err := otherConn.WriteMessage(gorillaws.TextMessage, []byte(`{"type":"get-conversations","payload":{}}`)); err != nil {
		t.Fatalf("expected the new connection to accept writes: %v", err)
	}
	otherConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := otherConn.ReadMessage(); err != nil {
		t.Fatalf("expected the server to still respond normally on a new connection after the panic: %v", err)
	}
}

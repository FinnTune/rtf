package websocket_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

// TestLogoutHandler_ConcurrentWithCheckLogin_NoDataRace fires LogoutHandler
// and CheckLoginHandler at the same session concurrently. checkLogin reads
// and writes Client.loggedIn while holding the manager's full write lock for
// its entire call; serveLogout must do the same (rather than writing it
// after releasing its lookup RLock) or go test -race flags an unsynchronized
// concurrent read/write of that field. Only meaningful run with -race — see
// TestAuthenticatedClientFromRequest_ConcurrentRequestsForSameSession for the
// same rationale.
func TestLogoutHandler_ConcurrentWithCheckLogin_NoDataRace(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	db.SetMaxOpenConns(1)

	const trials = 50
	for trial := 0; trial < trials; trial++ {
		sessionID := fmt.Sprintf("session-logout-race-%d", trial)
		websocket.AddAuthenticatedClient(sessionID, "alice", 2)
		websocket.SetLoggedInList("alice")

		var wg sync.WaitGroup
		wg.Add(4)

		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewBufferString("{}"))
			req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
			websocket.LogoutHandler(httptest.NewRecorder(), req)
		}()

		for i := 0; i < 3; i++ {
			go func() {
				defer wg.Done()
				req := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
				req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
				websocket.CheckLoginHandler(httptest.NewRecorder(), req)
			}()
		}

		wg.Wait()
	}
}

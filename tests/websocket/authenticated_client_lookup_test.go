package websocket_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

func addPostRequest(t *testing.T, sessionID, title string) *http.Request {
	t.Helper()
	payload := map[string]any{
		"title":   title,
		"content": "content",
		"categories": []map[string]any{
			{"id": 1, "name": "Code"},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/addPost", bytes.NewBuffer(data))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	return req
}

// TestAuthenticatedClientFromRequest_ExpiredSessionRemovedFromManager covers
// authenticatedClientFromRequest's rarer path — the RLock-based common-case
// scan finds an expired client, then the handler must escalate to a write
// lock to actually remove it from manager.clients, not just reject this one
// request. Exercised through a real handler (AddPost) rather than calling
// the unexported function directly, since that's the only way it's reached
// in production.
func TestAuthenticatedClientFromRequest_ExpiredSessionRemovedFromManager(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	handle := websocket.AddTestClient("session-expired", "actual_user", 42)
	handle.ExpireForTest()
	if got := websocket.ClientCountForTest(); got != 1 {
		t.Fatalf("expected 1 pre-registered client, got %d", got)
	}

	rr := httptest.NewRecorder()
	websocket.AddPost(rr, addPostRequest(t, "session-expired", "should be rejected"))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}
	if got := websocket.ClientCountForTest(); got != 0 {
		t.Fatalf("expected the expired client to be removed from the manager, got %d clients remaining", got)
	}
}

// TestAuthenticatedClientFromRequest_ConcurrentRequestsForSameSession fires
// many authenticated requests for the same session at once — the exact
// shape of concurrent traffic authenticatedClientFromRequest's RLock-based
// scan (plus Client.lastSeen's lock-free touch()) needs to handle safely.
// Only meaningful run with -race: it exercises concurrent touch() calls and
// concurrent map reads racing the (absent, in this test) sweep/expiry
// delete path.
func TestAuthenticatedClientFromRequest_ConcurrentRequestsForSameSession(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	// database/sql's default unlimited pool would hand concurrent goroutines
	// separate connections — and a bare ":memory:" DSN (no shared cache)
	// gives each connection its own isolated database, silently defeating
	// this test (every goroutine would see an empty schema). Forcing a
	// single shared connection is what actually lets the goroutines below
	// interleave against one real database, the way concurrent requests
	// through the production connection pool do. Same pattern as
	// conversations_race_test.go's TestOpenDirectChat_ConcurrentFirstMessageIsRaceSafe.
	db.SetMaxOpenConns(1)

	websocket.AddAuthenticatedClient("session-concurrent", "actual_user", 42)

	const n = 30
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rr := httptest.NewRecorder()
			websocket.AddPost(rr, addPostRequest(t, "session-concurrent", fmt.Sprintf("post %d", i)))
			codes[i] = rr.Code
		}(i)
	}
	wg.Wait()

	for i, code := range codes {
		if code != http.StatusOK {
			t.Fatalf("request %d: expected status %d, got %d", i, http.StatusOK, code)
		}
	}
	if got := websocket.ClientCountForTest(); got != 1 {
		t.Fatalf("expected exactly 1 client throughout, got %d", got)
	}
}

package websocket_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

func setUserBannedRequest(t *testing.T, userID int, banned bool) *bytes.Buffer {
	t.Helper()
	body, err := json.Marshal(map[string]any{"user_id": userID, "banned": banned})
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	return bytes.NewBuffer(body)
}

func TestListUsersHandler_ReturnsAllUsersWithoutPasswordHash(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/listUsers", nil)
	rr := httptest.NewRecorder()

	websocket.ListUsersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var users []struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Banned   bool   `json:"banned"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&users); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 seeded users, got %d: %+v", len(users), users)
	}
	if rr.Body.String() != "" && bytes.Contains(rr.Body.Bytes(), []byte("password")) {
		t.Fatal("response should never include a password field")
	}
}

func TestSetUserBannedHandler_BansAndUnbansUser(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	req := httptest.NewRequest(http.MethodPost, "/setUserBanned", setUserBannedRequest(t, 42, true))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.SetUserBannedHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var banned bool
	if err := db.QueryRow(`SELECT banned FROM user WHERE id = 42`).Scan(&banned); err != nil {
		t.Fatalf("failed to query banned status: %v", err)
	}
	if !banned {
		t.Fatal("expected user to be banned")
	}

	// Unban.
	req2 := httptest.NewRequest(http.MethodPost, "/setUserBanned", setUserBannedRequest(t, 42, false))
	req2.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr2 := httptest.NewRecorder()
	websocket.SetUserBannedHandler(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr2.Code, rr2.Body.String())
	}
	if err := db.QueryRow(`SELECT banned FROM user WHERE id = 42`).Scan(&banned); err != nil {
		t.Fatalf("failed to query banned status: %v", err)
	}
	if banned {
		t.Fatal("expected user to be unbanned")
	}
}

func TestSetUserBannedHandler_RejectsBanningSelf(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	req := httptest.NewRequest(http.MethodPost, "/setUserBanned", setUserBannedRequest(t, 1, true))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.SetUserBannedHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
	var banned bool
	if err := db.QueryRow(`SELECT banned FROM user WHERE id = 1`).Scan(&banned); err != nil {
		t.Fatalf("failed to query banned status: %v", err)
	}
	if banned {
		t.Fatal("admin should not have been able to ban themselves")
	}
}

func TestSetUserBannedHandler_DisconnectsLiveClientImmediately(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)
	target := websocket.AddTestClient("session-target", "actual_user", 42)
	websocket.SetLoggedInList("actual_user")

	req := httptest.NewRequest(http.MethodPost, "/setUserBanned", setUserBannedRequest(t, 42, true))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.SetUserBannedHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if !target.IsRemovedFromManager() {
		t.Fatal("expected the banned user's live client to be removed from the manager")
	}
	if websocket.IsInLoggedInList("actual_user") {
		t.Fatal("expected the banned user to be removed from LoggedInList")
	}
}

func TestSetUserBannedHandler_RejectsNonAdmin(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	handler := websocket.RequireAdmin(websocket.SetUserBannedHandler)
	req := httptest.NewRequest(http.MethodPost, "/setUserBanned", setUserBannedRequest(t, 42, true))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-alice"})
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}
}

func TestLoginHandler_RejectsBannedUser(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	if _, err := db.Exec(`UPDATE user SET banned = 1 WHERE uname = 'alice'`); err != nil {
		t.Fatalf("failed to ban alice: %v", err)
	}

	body := `{"username":"alice","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.LoginHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}
}

func TestCheckLoginHandler_TerminatesSessionForBannedUser(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-banned", "alice", 2)

	if _, err := db.Exec(`UPDATE user SET banned = 1 WHERE uname = 'alice'`); err != nil {
		t.Fatalf("failed to ban alice: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-banned"})
	rr := httptest.NewRecorder()

	websocket.CheckLoginHandler(rr, req)

	var resp websocket.UserLoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.LoggedIn {
		t.Fatal("expected a banned user's checkLogin poll to report loggedIn false")
	}

	handle := websocket.FindClientBySessionForTest("session-banned")
	if handle != nil {
		t.Fatal("expected the banned user's client to be removed from the manager")
	}
}

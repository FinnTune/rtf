package websocket_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"testing"
)

func TestRegistrationHandler_Success(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	body := `{
		"fname":"Bob","lname":"Jones","uname":"bob","email":"bob@example.com",
		"age":"31","gender":"male","password":"newpass123"
	}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.RegistrationHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM user WHERE uname = ?`, "bob").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 registered user, got %d", count)
	}
}

func TestRegistrationHandler_InvalidJSON(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{invalid`))
	rr := httptest.NewRecorder()

	websocket.RegistrationHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestLoginHandler_Success(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"username":"alice","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.LoginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp websocket.UserLoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.OTP == "" {
		t.Fatal("expected OTP in login response")
	}
	if resp.Username != "alice" {
		t.Fatalf("expected username alice, got %q", resp.Username)
	}
	if resp.ID != 2 {
		t.Fatalf("expected user id 2, got %d", resp.ID)
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"username":"alice","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.LoginHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestLoginHandler_UserNotFound(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"username":"nobody","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.LoginHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestCheckLoginHandler_NoCookie(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
	rr := httptest.NewRecorder()

	websocket.CheckLoginHandler(rr, req)

	var resp websocket.UserLoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.LoggedIn {
		t.Fatal("expected loggedIn false without session cookie")
	}
}

func TestCheckLoginHandler_WithAuthenticatedClient(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	websocket.AddAuthenticatedClient("session-check", "alice", 2)

	req := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-check"})
	rr := httptest.NewRecorder()

	websocket.CheckLoginHandler(rr, req)

	var resp websocket.UserLoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.LoggedIn {
		t.Fatal("expected loggedIn true for authenticated client")
	}
	if resp.Username != "alice" {
		t.Fatalf("expected username alice, got %q", resp.Username)
	}
	if resp.OTP == "" {
		t.Fatal("expected OTP for websocket connection")
	}
}

func TestLogoutHandler_LogsOutClient(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	client := websocket.AddTestClient("session-logout", "alice", 2)
	websocket.SetLoggedInList("alice")

	body := bytes.NewBufferString("{}")
	req := httptest.NewRequest(http.MethodPost, "/logout", body)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-logout"})
	rr := httptest.NewRecorder()

	websocket.LogoutHandler(rr, req)

	var resp websocket.UserLoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.LoggedIn {
		t.Fatal("expected loggedIn false after logout")
	}
	if websocket.IsInLoggedInList("alice") {
		t.Fatal("expected alice removed from LoggedInList")
	}
	if !client.IsRemovedFromManager() {
		t.Fatal("expected client removed from manager")
	}
}

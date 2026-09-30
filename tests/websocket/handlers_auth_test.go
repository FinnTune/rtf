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

// TestRegistrationHandler_NormalizesEmailToLowercase guards the fix for
// email case-sensitivity: without it, "Bob@Example.com" and
// "bob@example.com" would be treated as two distinct, independently
// registerable emails despite user.email's UNIQUE constraint.
func TestRegistrationHandler_NormalizesEmailToLowercase(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	body := `{
		"fname":"Bob","lname":"Jones","uname":"bob","email":"Bob@Example.com",
		"age":"31","gender":"male","password":"newpass123"
	}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.RegistrationHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var email string
	if err := db.QueryRow(`SELECT email FROM user WHERE uname = 'bob'`).Scan(&email); err != nil {
		t.Fatalf("failed to fetch registered email: %v", err)
	}
	if email != "bob@example.com" {
		t.Fatalf("expected the stored email to be lowercased, got %q", email)
	}
}

// TestRegistrationHandler_RejectsEmailDifferingOnlyByCaseFromExisting
// guards the other half of the same fix: a new registration must be
// rejected as a duplicate of an existing account's email even when the
// only difference is case (the seeded alice already owns
// alice@example.com).
func TestRegistrationHandler_RejectsEmailDifferingOnlyByCaseFromExisting(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{
		"fname":"Second","lname":"Alice","uname":"alice2","email":"Alice@Example.com",
		"age":"31","gender":"female","password":"newpass123"
	}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.RegistrationHandler(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rr.Code, rr.Body.String())
	}
}

// TestRegistrationHandler_RejectsEmailDifferingOnlyByCaseFromLegacyMixedCaseRow
// guards the stored-side LOWER(email) comparison specifically: the
// seeded test users' emails are already lowercase, so
// TestRegistrationHandler_RejectsEmailDifferingOnlyByCaseFromExisting alone
// wouldn't catch a regression that only lowercases the newly-submitted
// email (validateRegistration) without also comparing case-insensitively
// against whatever case an EXISTING row happens to be stored in — exactly
// the shape of an older, pre-normalization row (see migrate()'s
// normalizeExistingEmailsToLowercase, which backfills these where safe,
// but can't guarantee every row ever gets backfilled).
func TestRegistrationHandler_RejectsEmailDifferingOnlyByCaseFromLegacyMixedCaseRow(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	if _, err := db.Exec(
		`INSERT INTO user (fname, lname, uname, email, age, gender, pass, created_at) VALUES
		('Legacy', 'User', 'legacyuser', 'Legacy@Example.com', '40', 'other', 'hash', datetime('now'))`,
	); err != nil {
		t.Fatalf("failed to seed a legacy mixed-case email row: %v", err)
	}

	body := `{
		"fname":"New","lname":"User","uname":"newuser","email":"legacy@example.com",
		"age":"31","gender":"male","password":"newpass123"
	}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.RegistrationHandler(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rr.Code, rr.Body.String())
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
	if resp.Role != "user" {
		t.Fatalf("expected role 'user' for alice, got %q", resp.Role)
	}
}

// TestLoginHandler_MatchesEmailCaseInsensitively guards the login half of
// the email case-sensitivity fix: the seeded alice's email is
// alice@example.com, but logging in with any other casing must still work
// — a user who registered with (or whose email was later normalized to a
// different case than) what they type shouldn't be locked out.
func TestLoginHandler_MatchesEmailCaseInsensitively(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"username":"ALICE@EXAMPLE.COM","password":"secret123"}`
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
	if resp.Username != "alice" {
		t.Fatalf("expected username alice, got %q", resp.Username)
	}
}

// TestLoginHandler_UsernameMatchingStaysCaseSensitive guards against the
// fix overreaching: usernames are case-preserving identity strings, unlike
// email — only the email side of serveLogin's lookup was made
// case-insensitive, and this confirms logging in with the wrong-case
// username still fails exactly like before.
func TestLoginHandler_UsernameMatchingStaysCaseSensitive(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"username":"ALICE","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.LoginHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected wrong-case username to still be rejected with status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestLoginHandler_ReturnsAdminRoleForAdminUser(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"username":"admin","password":"secret123"}`
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
	if resp.Role != "admin" {
		t.Fatalf("expected role 'admin' for the seeded admin user, got %q", resp.Role)
	}
}

func TestLoginHandler_BindsSessionToVerifiedIdentity(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"username":"alice","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.LoginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_id" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected a session_id cookie from login")
	}

	// The identity binding must already exist right after login — a
	// websocket never has to connect (and send a client-trusted
	// user-connect event) for the session to be recognized as alice.
	checkReq := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
	checkReq.AddCookie(sessionCookie)
	checkRR := httptest.NewRecorder()

	websocket.CheckLoginHandler(checkRR, checkReq)

	var resp websocket.UserLoginResponse
	if err := json.NewDecoder(checkRR.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode checkLogin response: %v", err)
	}
	if !resp.LoggedIn {
		t.Fatal("expected the new session to already be recognized as logged in, before any websocket connects")
	}
	if resp.Username != "alice" {
		t.Fatalf("expected username alice, got %q", resp.Username)
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
	if resp.Role != "user" {
		t.Fatalf("expected role 'user' for alice, got %q", resp.Role)
	}
}

func TestCheckLoginHandler_ReturnsAdminRoleForAdminUser(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	websocket.AddAuthenticatedClient("session-check-admin", "admin", 1)

	req := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-check-admin"})
	rr := httptest.NewRecorder()

	websocket.CheckLoginHandler(rr, req)

	var resp websocket.UserLoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Role != "admin" {
		t.Fatalf("expected role 'admin' for the seeded admin user, got %q", resp.Role)
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

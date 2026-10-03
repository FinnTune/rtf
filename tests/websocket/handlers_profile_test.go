package websocket_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"rtForum/tests/testutil"
	"rtForum/utility"
	"rtForum/websocket"
)

func authenticatedRequest(method, target string, body *bytes.Buffer, sessionID string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, body)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	return req
}

func TestGetProfileHandler_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	rr := httptest.NewRecorder()

	websocket.GetProfileHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestGetProfileHandler_ReturnsAuthenticatedUsersOwnProfile(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	req := authenticatedRequest(http.MethodGet, "/profile", nil, "session-123")
	rr := httptest.NewRecorder()

	websocket.GetProfileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var got struct {
		Fname string `json:"fname"`
		Lname string `json:"lname"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got.Fname != "Actual" || got.Lname != "User" || got.Email != "actual@example.com" {
		t.Fatalf("expected the seeded actual_user's own profile, got %+v", got)
	}
}

func TestUpdateProfileHandler_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body, _ := json.Marshal(map[string]string{"fname": "New", "lname": "Name", "email": "new@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	websocket.UpdateProfileHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestUpdateProfileHandler_UpdatesOwnFnameLnameAndEmail(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	handle := websocket.AddTestClient("session-123", "actual_user", 42)

	body, _ := json.Marshal(map[string]string{"fname": "Updated", "lname": "Person", "email": "updated@example.com"})
	req := authenticatedRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdateProfileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var fname, lname, email string
	if err := db.QueryRow(`SELECT fname, lname, email FROM user WHERE id = 42`).Scan(&fname, &lname, &email); err != nil {
		t.Fatalf("failed to fetch updated profile: %v", err)
	}
	if fname != "Updated" || lname != "Person" || email != "updated@example.com" {
		t.Fatalf("expected the profile to be updated in the database, got fname=%q lname=%q email=%q", fname, lname, email)
	}

	// checkLogin serves client.email straight from memory (see
	// (*Manager).checkLogin) — without updating it here too, the new email
	// wouldn't show up until the next full login even though the database
	// itself is already correct.
	if got := handle.Email(); got != "updated@example.com" {
		t.Fatalf("expected the in-memory client's cached email to be updated too, got %q", got)
	}
}

// TestUpdateProfileHandler_NormalizesEmailToLowercase guards the same fix
// as TestRegistrationHandler_NormalizesEmailToLowercase, for the profile
// update path.
func TestUpdateProfileHandler_NormalizesEmailToLowercase(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	body, _ := json.Marshal(map[string]string{"fname": "Actual", "lname": "User", "email": "New@Example.com"})
	req := authenticatedRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdateProfileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var email string
	if err := db.QueryRow(`SELECT email FROM user WHERE id = 42`).Scan(&email); err != nil {
		t.Fatalf("failed to fetch updated email: %v", err)
	}
	if email != "new@example.com" {
		t.Fatalf("expected the stored email to be lowercased, got %q", email)
	}
}

// TestUpdateProfileHandler_RejectsEmailDifferingOnlyByCaseFromAnotherUser
// guards the other half of the same fix: a profile update must be rejected
// as a duplicate of another user's email even when the only difference is
// case (alice, seeded, already owns alice@example.com).
func TestUpdateProfileHandler_RejectsEmailDifferingOnlyByCaseFromAnotherUser(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	body, _ := json.Marshal(map[string]string{"fname": "Actual", "lname": "User", "email": "Alice@Example.com"})
	req := authenticatedRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdateProfileHandler(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rr.Code, rr.Body.String())
	}
}

// TestUpdateProfileHandler_RejectsEmailDifferingOnlyByCaseFromLegacyMixedCaseRow
// guards the stored-side LOWER(email) comparison specifically — see the
// identical-in-spirit registration test's doc comment for why the
// seeded-alice test above alone doesn't cover this (her email is already
// lowercase).
func TestUpdateProfileHandler_RejectsEmailDifferingOnlyByCaseFromLegacyMixedCaseRow(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	if _, err := db.Exec(
		`INSERT INTO user (fname, lname, uname, email, age, gender, pass, created_at) VALUES
		('Legacy', 'User', 'legacyuser', 'Legacy@Example.com', '40', 'other', 'hash', datetime('now'))`,
	); err != nil {
		t.Fatalf("failed to seed a legacy mixed-case email row: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"fname": "Actual", "lname": "User", "email": "legacy@example.com"})
	req := authenticatedRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdateProfileHandler(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rr.Code, rr.Body.String())
	}
}

// TestUpdateProfileHandler_RejectsEmailAlreadyInUseByAnotherUser guards the
// same email-uniqueness invariant registration already enforces (user.email
// is UNIQUE) — a profile update must re-check it too, not just at signup.
func TestUpdateProfileHandler_RejectsEmailAlreadyInUseByAnotherUser(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	// alice (seeded, id 2) already owns alice@example.com.
	body, _ := json.Marshal(map[string]string{"fname": "Actual", "lname": "User", "email": "alice@example.com"})
	req := authenticatedRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdateProfileHandler(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rr.Code, rr.Body.String())
	}

	var email string
	if err := db.QueryRow(`SELECT email FROM user WHERE id = 42`).Scan(&email); err != nil {
		t.Fatalf("failed to fetch profile: %v", err)
	}
	if email != "actual@example.com" {
		t.Fatalf("expected the rejected update to leave the original email untouched, got %q", email)
	}
}

// TestUpdateProfileHandler_AllowsKeepingYourOwnCurrentEmail guards against
// an over-eager uniqueness check rejecting a no-op email "change" back to
// the same address the account already owns (e.g. a form re-submit that
// only actually changed fname/lname).
func TestUpdateProfileHandler_AllowsKeepingYourOwnCurrentEmail(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	body, _ := json.Marshal(map[string]string{"fname": "Actual", "lname": "Renamed", "email": "actual@example.com"})
	req := authenticatedRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdateProfileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestUpdateProfileHandler_RejectsInvalidEmail(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	body, _ := json.Marshal(map[string]string{"fname": "Actual", "lname": "User", "email": "not-an-email"})
	req := authenticatedRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdateProfileHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestUpdateProfileHandler_ConcurrentEmailRace_OnlyOneSucceeds guards the
// UNIQUE-constraint backstop below the COUNT(*) pre-check: two different
// users racing to claim the same brand-new email (neither owns it yet, so
// both pass the pre-check) must resolve to exactly one 200 and the rest
// clean 409s — never a 500, and never two rows sharing the email. Same
// shared-single-connection pattern as
// TestReactToPostHandler_ConcurrentInsertRace_NoServerErrors.
func TestUpdateProfileHandler_ConcurrentEmailRace_OnlyOneSucceeds(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	db.SetMaxOpenConns(1)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	sessions := []string{"session-alice", "session-actual"}
	var wg sync.WaitGroup
	codes := make([]int, len(sessions))
	bodies := make([]string, len(sessions))
	for i, sessionID := range sessions {
		wg.Add(1)
		go func(i int, sessionID string) {
			defer wg.Done()
			body, _ := json.Marshal(map[string]string{"fname": "Racer", "lname": "User", "email": "raceclaimed@example.com"})
			req := authenticatedRequest(http.MethodPost, "/updateProfile", bytes.NewBuffer(body), sessionID)
			rr := httptest.NewRecorder()
			websocket.UpdateProfileHandler(rr, req)
			codes[i] = rr.Code
			bodies[i] = rr.Body.String()
		}(i, sessionID)
	}
	wg.Wait()

	var successes, conflicts int
	for i, code := range codes {
		switch code {
		case http.StatusOK:
			successes++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("request %d: expected %d or %d, got %d: %s", i, http.StatusOK, http.StatusConflict, code, bodies[i])
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected exactly 1 success and 1 conflict, got %d successes and %d conflicts", successes, conflicts)
	}

	var rowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user WHERE email = 'raceclaimed@example.com'`).Scan(&rowCount); err != nil {
		t.Fatalf("failed to count rows with the raced email: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("expected exactly 1 row with the raced email, got %d — the UNIQUE constraint was violated", rowCount)
	}
}

// TestUpdatePasswordHandler_KicksAllSessionsForThisUser guards the one
// self-service remediation a user has after suspecting a session was
// compromised (a stolen-but-still-valid session_id cookie, the standard
// reason to change a password at all): changing it must invalidate every
// live session for this account, including a second tab/device (standing
// in for an attacker's stolen cookie) and the request's own current
// session — mirrors
// TestSetUserBannedHandler_DisconnectsLiveClientImmediately's shape, same
// kickUser mechanism.
func TestUpdatePasswordHandler_KicksAllSessionsForThisUser(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	owner := websocket.AddTestClient("session-owner", "actual_user", 42)
	otherDevice := websocket.AddTestClient("session-other-device", "actual_user", 42)
	websocket.SetLoggedInList("actual_user")

	body, _ := json.Marshal(map[string]string{"current_password": "secret123", "new_password": "newpassword1"})
	req := authenticatedRequest(http.MethodPost, "/updatePassword", bytes.NewBuffer(body), "session-owner")
	rr := httptest.NewRecorder()

	websocket.UpdatePasswordHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if !owner.IsRemovedFromManager() {
		t.Fatal("expected the request's own session to be removed from the manager too, not just other sessions")
	}
	if !otherDevice.IsRemovedFromManager() {
		t.Fatal("expected a second session for this same user (a stolen cookie, another tab) to be removed from the manager")
	}
	if websocket.IsInLoggedInList("actual_user") {
		t.Fatal("expected the user to be removed from LoggedInList")
	}
}

func TestUpdatePasswordHandler_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body, _ := json.Marshal(map[string]string{"current_password": "secret123", "new_password": "newpassword1"})
	req := httptest.NewRequest(http.MethodPost, "/updatePassword", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	websocket.UpdatePasswordHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestUpdatePasswordHandler_RejectsWrongCurrentPassword guards the one
// thing standing between a stolen-but-still-valid session cookie and an
// attacker locking the real account owner out by changing their password.
func TestUpdatePasswordHandler_RejectsWrongCurrentPassword(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	body, _ := json.Marshal(map[string]string{"current_password": "wrong-password", "new_password": "newpassword1"})
	req := authenticatedRequest(http.MethodPost, "/updatePassword", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdatePasswordHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	var hash string
	if err := db.QueryRow(`SELECT pass FROM user WHERE id = 42`).Scan(&hash); err != nil {
		t.Fatalf("failed to fetch password hash: %v", err)
	}
	if !utility.CheckPasswordHash("secret123", hash) {
		t.Fatal("expected the original password to still work after a rejected change")
	}
}

func TestUpdatePasswordHandler_RejectsTooShortNewPassword(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	body, _ := json.Marshal(map[string]string{"current_password": "secret123", "new_password": "short"})
	req := authenticatedRequest(http.MethodPost, "/updatePassword", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdatePasswordHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestUpdatePasswordHandler_UpdatesPasswordSuccessfully(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	body, _ := json.Marshal(map[string]string{"current_password": "secret123", "new_password": "newpassword1"})
	req := authenticatedRequest(http.MethodPost, "/updatePassword", bytes.NewBuffer(body), "session-123")
	rr := httptest.NewRecorder()

	websocket.UpdatePasswordHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var hash string
	if err := db.QueryRow(`SELECT pass FROM user WHERE id = 42`).Scan(&hash); err != nil {
		t.Fatalf("failed to fetch password hash: %v", err)
	}
	if !utility.CheckPasswordHash("newpassword1", hash) {
		t.Fatal("expected the new password to work after a successful change")
	}
	if utility.CheckPasswordHash("secret123", hash) {
		t.Fatal("expected the old password to no longer work after a successful change")
	}
}

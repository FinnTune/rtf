package websocket_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	if got := rr.Header().Get("X-Total-Count"); got != "3" {
		t.Fatalf("expected X-Total-Count 3, got %q", got)
	}
}

// TestListUsersHandler_RespectsLimitAndOffset is the regression test for
// ListUsersHandler's pagination: it used to run an unbounded SELECT with no
// LIMIT/OFFSET at all, unlike every other listing endpoint in this file
// (posts, comments, search). Mirrors AllPostsHandler's own limit/offset
// tests.
func TestListUsersHandler_RespectsLimitAndOffset(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/listUsers?limit=1&offset=1", nil)
	rr := httptest.NewRecorder()
	websocket.ListUsersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var users []struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&users); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected exactly 1 user with limit=1, got %d: %+v", len(users), users)
	}
	if got := rr.Header().Get("X-Total-Count"); got != "3" {
		t.Fatalf("expected X-Total-Count 3 regardless of the page size, got %q", got)
	}
}

func TestListUsersHandler_CapsLimitAtMax(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	for i := 0; i < 120; i++ {
		_, err := db.Exec(
			`INSERT INTO user (fname, lname, uname, email, age, gender, pass, created_at) VALUES
			('bulk', 'user', ?, ?, '30', 'other', 'hash', datetime('now'))`,
			fmt.Sprintf("bulkuser%d", i), fmt.Sprintf("bulkuser%d@example.com", i),
		)
		if err != nil {
			t.Fatalf("failed to seed bulk user: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/listUsers?limit=9999", nil)
	rr := httptest.NewRecorder()
	websocket.ListUsersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var users []struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&users); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(users) != 100 {
		t.Fatalf("expected limit to be capped at 100, got %d users", len(users))
	}
	if got := rr.Header().Get("X-Total-Count"); got != "123" {
		t.Fatalf("expected X-Total-Count 123 (3 seeded + 120 bulk), got %q", got)
	}
}

// TestListUsersHandler_FiltersByUsernameOrEmailSubstring is the regression
// test for the actual fix: ?q= used to not exist at all, so an admin
// looking for one user among many had no way to do so besides paging
// through the whole list alphabetically.
func TestListUsersHandler_FiltersByUsernameOrEmailSubstring(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/listUsers?q=ali", nil)
	rr := httptest.NewRecorder()
	websocket.ListUsersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var users []struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&users); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(users) != 1 || users[0].Username != "alice" {
		t.Fatalf("expected exactly alice to match %q, got %+v", "ali", users)
	}
	if got := rr.Header().Get("X-Total-Count"); got != "1" {
		t.Fatalf("expected X-Total-Count to reflect the filtered set (1), got %q", got)
	}
}

// TestListUsersHandler_FilterMatchesEmailToo guards the OR half of the
// filter — a query matching only the email, not the username, must still
// find the user.
func TestListUsersHandler_FilterMatchesEmailToo(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// actual_user's seeded email is actual@example.com — "actual@" doesn't
	// appear in the username "actual_user" itself, only the email.
	req := httptest.NewRequest(http.MethodGet, "/listUsers?q=actual@", nil)
	rr := httptest.NewRecorder()
	websocket.ListUsersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var users []struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&users); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(users) != 1 || users[0].Username != "actual_user" {
		t.Fatalf("expected exactly actual_user to match by email, got %+v", users)
	}
}

// TestListUsersHandler_FilterIsCaseInsensitive guards against a regression
// this codebase has hit before (see the email/category case-sensitivity
// fixes) — SQLite's LIKE is case-insensitive by default for ASCII, which
// uname/email always are, but this pins that behavior down explicitly.
func TestListUsersHandler_FilterIsCaseInsensitive(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/listUsers?q=ALICE", nil)
	rr := httptest.NewRecorder()
	websocket.ListUsersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var users []struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&users); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(users) != 1 || users[0].Username != "alice" {
		t.Fatalf("expected a case-insensitive match for alice, got %+v", users)
	}
}

// TestListUsersHandler_BlankQueryReturnsEveryone guards the "optional"
// half of the fix — unlike SearchPostsHandler's ?q= (which rejects an
// empty query, since search is that endpoint's whole purpose), an empty
// or whitespace-only q here must behave exactly like no filter at all.
func TestListUsersHandler_BlankQueryReturnsEveryone(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/listUsers?q=%20%20", nil)
	rr := httptest.NewRecorder()
	websocket.ListUsersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("X-Total-Count"); got != "3" {
		t.Fatalf("expected all 3 seeded users with a blank query, got X-Total-Count %q", got)
	}
}

func TestListUsersHandler_RejectsOverlongQuery(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/listUsers?q="+strings.Repeat("a", 101), nil)
	rr := httptest.NewRecorder()
	websocket.ListUsersHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
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

// TestSetUserBannedHandler_RejectsBanningAnotherAdmin guards against a real
// governance/lockout risk: RequireAdmin gates this whole handler, but
// nothing beyond that stopped one admin from banning a DIFFERENT admin
// account — kickUser disconnects them immediately and checkLogin/serveLogin
// reject them from then on, with no role-hierarchy or "last admin" concept
// to fall back on.
func TestSetUserBannedHandler_RejectsBanningAnotherAdmin(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	if _, err := db.Exec(`UPDATE user SET role = 'admin' WHERE id = 2`); err != nil {
		t.Fatalf("failed to promote alice to admin: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/setUserBanned", setUserBannedRequest(t, 2, true))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.SetUserBannedHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
	var banned bool
	if err := db.QueryRow(`SELECT banned FROM user WHERE id = 2`).Scan(&banned); err != nil {
		t.Fatalf("failed to query banned status: %v", err)
	}
	if banned {
		t.Fatal("an admin should not have been able to ban a fellow admin")
	}
}

// TestSetUserBannedHandler_AllowsUnbanningAnotherAdmin guards against the
// fix overreaching: only the BAN direction is a governance risk — an
// already-banned admin account (e.g. banned before ever being promoted,
// or by a since-corrected mistake) must still be unbannable.
func TestSetUserBannedHandler_AllowsUnbanningAnotherAdmin(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	if _, err := db.Exec(`UPDATE user SET role = 'admin', banned = 1 WHERE id = 2`); err != nil {
		t.Fatalf("failed to set up a banned fellow admin: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/setUserBanned", setUserBannedRequest(t, 2, false))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.SetUserBannedHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var banned bool
	if err := db.QueryRow(`SELECT banned FROM user WHERE id = 2`).Scan(&banned); err != nil {
		t.Fatalf("failed to query banned status: %v", err)
	}
	if banned {
		t.Fatal("expected the fellow admin to be unbanned")
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

// TestSetUserBannedHandler_BroadcastsUpdatedUsersList is the regression
// test for the presence-broadcast fix on the admin-kick path: kickUser
// used to remove the banned user from LoggedInList without ever telling
// anyone else — other connected clients' "online" sidebar only refreshed
// whenever some unrelated user happened to connect or disconnect next,
// which meant a freshly-banned, disruptive user could still appear online
// to everyone indefinitely.
func TestSetUserBannedHandler_BroadcastsUpdatedUsersList(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	admin := websocket.AddTestClient("session-admin", "admin", 1)
	websocket.AddTestClient("session-target", "actual_user", 42)
	websocket.SetLoggedInList("actual_user")

	req := httptest.NewRequest(http.MethodPost, "/setUserBanned", setUserBannedRequest(t, 42, true))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.SetUserBannedHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	eventType, payload, ok := admin.WaitEvent(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for a users-online broadcast after the ban")
	}
	if eventType != websocket.UsersList {
		t.Fatalf("expected a %q broadcast, got %q", websocket.UsersList, eventType)
	}
	var snapshot map[string]bool
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		t.Fatalf("failed to decode users-online payload: %v", err)
	}
	if snapshot["actual_user"] {
		t.Fatalf("expected the banned user to no longer be in the broadcast users-online list, got %+v", snapshot)
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

// TestCheckLoginHandler_BannedUser_BroadcastsUpdatedUsersList is the
// regression test for the presence-broadcast fix on checkLogin's own ban
// path (distinct from the admin-kick path — this fires when a poll
// discovers a ban that happened since the user's last checkLogin, rather
// than an admin action against a currently-registered client). checkLogin
// holds the manager's write lock for its whole body, so the fix has to
// broadcast without re-acquiring it (see broadcastUsersListLocked) — this
// confirms that actually happens, not just that the deadlock was avoided.
func TestCheckLoginHandler_BannedUser_BroadcastsUpdatedUsersList(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-banned", "alice", 2)
	observer := websocket.AddTestClient("session-observer", "observer", 99)
	websocket.SetLoggedInList("alice")

	if _, err := db.Exec(`UPDATE user SET banned = 1 WHERE uname = 'alice'`); err != nil {
		t.Fatalf("failed to ban alice: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-banned"})
	rr := httptest.NewRecorder()

	websocket.CheckLoginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	eventType, payload, ok := observer.WaitEvent(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for a users-online broadcast after the ban")
	}
	if eventType != websocket.UsersList {
		t.Fatalf("expected a %q broadcast, got %q", websocket.UsersList, eventType)
	}
	var snapshot map[string]bool
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		t.Fatalf("failed to decode users-online payload: %v", err)
	}
	if snapshot["alice"] {
		t.Fatalf("expected the banned user to no longer be in the broadcast users-online list, got %+v", snapshot)
	}
}

package websocket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"testing"
)

type userProfileResponse struct {
	Username     string `json:"username"`
	Joined       string `json:"joined"`
	PostCount    int    `json:"post_count"`
	CommentCount int    `json:"comment_count"`
}

func TestGetUserProfileHandler_RejectsInvalidUsername(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getUserProfile?username=a", nil)
	rr := httptest.NewRecorder()

	websocket.GetUserProfileHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestGetUserProfileHandler_RejectsNonexistentUser(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getUserProfile?username=nobody_here", nil)
	rr := httptest.NewRecorder()

	websocket.GetUserProfileHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

// TestGetUserProfileHandler_WorksUnauthenticated guards that viewing a
// public profile needs no session, same as browsing anyone's posts already
// doesn't (GetPostsByAuthorHandler) — no session_id cookie is attached here
// at all.
func TestGetUserProfileHandler_WorksUnauthenticated(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getUserProfile?username=actual_user", nil)
	rr := httptest.NewRecorder()

	websocket.GetUserProfileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

// TestGetUserProfileHandler_ReturnsJoinedAndCounts covers the actual
// summary data: join date (matching the real seeded user row, not a
// placeholder), post count via post.author (seed post 1 belongs to
// actual_user), and comment count via comment.user_id — the seed data
// already has one comment (id 1) from actual_user on post 1; two more are
// added here (ids 2, 3, picking up where the seed left off) on OTHER
// posts, for a total of 3, confirming the count isn't accidentally scoped
// to a single post.
func TestGetUserProfileHandler_ReturnsJoinedAndCounts(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	var wantJoined string
	if err := db.QueryRow(`SELECT created_at FROM user WHERE uname = 'actual_user'`).Scan(&wantJoined); err != nil {
		t.Fatalf("failed to read seeded join date: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO comment (id, user_id, post_id, content, created_at) VALUES
		(2, 42, 2, 'second comment', datetime('now')),
		(3, 42, 3, 'third comment', datetime('now'));
	`); err != nil {
		t.Fatalf("failed to seed comments: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/getUserProfile?username=actual_user", nil)
	rr := httptest.NewRecorder()

	websocket.GetUserProfileHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var profile userProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &profile); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if profile.Username != "actual_user" {
		t.Fatalf("expected username %q, got %q", "actual_user", profile.Username)
	}
	if profile.Joined != wantJoined {
		t.Fatalf("expected joined %q, got %q", wantJoined, profile.Joined)
	}
	if profile.PostCount != 1 {
		t.Fatalf("expected post_count 1 (seed post 1), got %d", profile.PostCount)
	}
	if profile.CommentCount != 3 {
		t.Fatalf("expected comment_count 3 (1 seeded + 2 added), got %d", profile.CommentCount)
	}
}

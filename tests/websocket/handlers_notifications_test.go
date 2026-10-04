package websocket_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"testing"
	"time"
)

// addComment is a small shared helper for these tests — posts a comment as
// sessionID and fails the test unless it succeeds, mirroring the shape
// already used for reactToPost/similar helpers elsewhere in this package.
func addComment(t *testing.T, sessionID string, postID int, content string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"post_id": postID, "content": content})
	req := httptest.NewRequest(http.MethodPost, "/addcomment", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	rr := httptest.NewRecorder()
	websocket.AddCommentHandler(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("addComment: expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}
}

// TestAddCommentHandler_CreatesNotificationForPostOwner guards the core of
// the feature: commenting on someone else's post must leave a notification
// row for the post's owner, not the commenter — seed post 1 is owned by
// actual_user (id 42); alice (id 2) is the commenter here.
func TestAddCommentHandler_CreatesNotificationForPostOwner(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	addComment(t, "session-alice", 1, "nice post!")

	var userID, postID, commentID int
	var actorUsername string
	if err := db.QueryRow(`SELECT user_id, post_id, comment_id, actor_username FROM notification`).
		Scan(&userID, &postID, &commentID, &actorUsername); err != nil {
		t.Fatalf("expected exactly one notification row: %v", err)
	}
	if userID != 42 || postID != 1 || actorUsername != "alice" {
		t.Fatalf("expected {user_id:42 post_id:1 actor_username:alice}, got {user_id:%d post_id:%d actor_username:%q}", userID, postID, actorUsername)
	}
	if commentID <= 0 {
		t.Fatalf("expected a real comment_id, got %d", commentID)
	}
}

// TestAddCommentHandler_NoSelfNotificationOnOwnPost guards against a user
// notifying themselves — actual_user (42) commenting on their own seed
// post 1 must create no notification row at all.
func TestAddCommentHandler_NoSelfNotificationOnOwnPost(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	addComment(t, "session-owner", 1, "my own post, commenting on it myself")

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notification`).Scan(&count); err != nil {
		t.Fatalf("failed to count notifications: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no notification for commenting on your own post, got %d", count)
	}
}

// TestAddCommentHandler_PushesNotificationAddedToOwnersLiveClientOnly
// guards broadcastNotificationAdded's targeting: only the post owner's own
// live client(s) should receive notification-added — not the commenter,
// and not an unrelated third connected client.
func TestAddCommentHandler_PushesNotificationAddedToOwnersLiveClientOnly(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	owner := websocket.AddTestClient("session-owner", "actual_user", 42)
	bystander := websocket.AddTestClient("session-bystander", "admin", 1)

	addComment(t, "session-alice", 1, "nice post!")

	// The owner is also "every other connected client" as far as
	// broadcastCommentAdded is concerned, so comment-added arrives first —
	// drain it before the notification-added push that follows.
	if eventType, _, ok := owner.WaitEvent(2 * time.Second); !ok || eventType != websocket.CommentAdded {
		t.Fatalf("expected to drain a %q push first, got type=%q ok=%v", websocket.CommentAdded, eventType, ok)
	}

	eventType, payload, ok := owner.WaitEvent(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for a notification-added push to the post owner")
	}
	if eventType != websocket.NotificationAdded {
		t.Fatalf("expected a %q push, got %q", websocket.NotificationAdded, eventType)
	}
	var notif websocket.Notification
	if err := json.Unmarshal(payload, &notif); err != nil {
		t.Fatalf("failed to decode notification-added payload: %v", err)
	}
	if notif.PostID != 1 || notif.ActorUsername != "alice" || notif.Read {
		t.Fatalf("unexpected notification payload: %+v", notif)
	}

	// The bystander also receives comment-added (broadcastCommentAdded
	// goes to every other connected client, not just the post's owner) —
	// drain that expected push before asserting no notification-added
	// follows it for them.
	if eventType, _, ok := bystander.WaitEvent(2 * time.Second); !ok || eventType != websocket.CommentAdded {
		t.Fatalf("expected to drain a %q push first, got type=%q ok=%v", websocket.CommentAdded, eventType, ok)
	}
	if _, _, ok := bystander.WaitEvent(200 * time.Millisecond); ok {
		t.Fatal("expected an unrelated connected client not to receive the owner's notification push")
	}
}

func TestGetNotificationsHandler_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rr := httptest.NewRecorder()

	websocket.GetNotificationsHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestGetNotificationsHandler_ReturnsOwnNotificationsNewestFirstWithCounts
// covers ordering, the joined post title, and both count headers —
// including that marking one read is reflected in X-Unread-Count but not
// X-Total-Count.
func TestGetNotificationsHandler_ReturnsOwnNotificationsNewestFirstWithCounts(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	// Two different commenters on actual_user's own posts (1, owned by
	// actual_user) and (2/3, owned by admin) — only post 1 notifies
	// actual_user; comments on posts 2/3 by admin notify nobody relevant
	// here. Use alice commenting on post 1 twice instead, for two
	// notifications to the SAME owner in a known order.
	addComment(t, "session-alice", 1, "first comment")
	addComment(t, "session-alice", 1, "second comment")

	// actual_user (post 1's owner) isn't connected above — log in a plain
	// authenticated client for the request itself.
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.GetNotificationsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("X-Total-Count"); got != "2" {
		t.Fatalf("expected X-Total-Count 2, got %q", got)
	}

	var response struct {
		Notifications []websocket.Notification `json:"notifications"`
		UnreadCount   int                       `json:"unread_count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.UnreadCount != 2 {
		t.Fatalf("expected unread_count 2, got %d", response.UnreadCount)
	}
	notifications := response.Notifications
	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifications))
	}
	if notifications[0].CommentID == notifications[1].CommentID {
		t.Fatal("expected two distinct comment ids")
	}
	// newest first.
	if notifications[0].ID <= notifications[1].ID {
		t.Fatalf("expected newest-first order, got ids %d then %d", notifications[0].ID, notifications[1].ID)
	}
	for _, n := range notifications {
		if n.PostTitle != "seed" {
			t.Fatalf("expected the joined post title %q, got %q", "seed", n.PostTitle)
		}
		if n.Read {
			t.Fatal("expected both notifications to start unread")
		}
	}
}

func TestMarkNotificationsReadHandler_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/markNotificationsRead", nil)
	rr := httptest.NewRecorder()

	websocket.MarkNotificationsReadHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestMarkNotificationsReadHandler_MarksAllWhenNoIDGiven(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	addComment(t, "session-alice", 1, "first")
	addComment(t, "session-alice", 1, "second")

	req := httptest.NewRequest(http.MethodPost, "/markNotificationsRead", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.MarkNotificationsReadHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var unread int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notification WHERE user_id = 42 AND read_at IS NULL`).Scan(&unread); err != nil {
		t.Fatalf("failed to count unread: %v", err)
	}
	if unread != 0 {
		t.Fatalf("expected 0 unread after marking all read, got %d", unread)
	}
}

// TestMarkNotificationsReadHandler_CannotMarkAnotherUsersNotification
// guards the WHERE user_id = ? scoping: an id belonging to someone else's
// notification must silently no-op (not error — see the handler's own doc
// comment on why it never reveals whether that id exists at all), leaving
// the real owner's notification untouched.
func TestMarkNotificationsReadHandler_CannotMarkAnotherUsersNotification(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-attacker", "admin", 1)

	addComment(t, "session-alice", 1, "first")

	var notifID int
	if err := db.QueryRow(`SELECT id FROM notification`).Scan(&notifID); err != nil {
		t.Fatalf("failed to find seeded notification id: %v", err)
	}

	body, _ := json.Marshal(map[string]int{"id": notifID})
	req := httptest.NewRequest(http.MethodPost, "/markNotificationsRead", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-attacker"})
	rr := httptest.NewRecorder()

	websocket.MarkNotificationsReadHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d (silent no-op), got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var readAt *string
	if err := db.QueryRow(`SELECT read_at FROM notification WHERE id = ?`, notifID).Scan(&readAt); err != nil {
		t.Fatalf("failed to check read_at: %v", err)
	}
	if readAt != nil {
		t.Fatal("expected the real owner's notification to remain unread after another user's mark-read attempt")
	}
}

// TestDeletePostHandler_RemovesDependentNotifications guards against the
// new notification.post_id/comment_id foreign keys blocking post deletion
// outright (foreign keys are enforced on this connection) once a post has
// a notification pointing at it.
func TestDeletePostHandler_RemovesDependentNotifications(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	addComment(t, "session-alice", 1, "nice post!")

	req := httptest.NewRequest(http.MethodPost, "/deletePost", bytes.NewBufferString(`{"id":1}`))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.DeletePostHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notification WHERE post_id = 1`).Scan(&count); err != nil {
		t.Fatalf("failed to count notifications: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the post's notifications to be deleted along with it, got %d", count)
	}
}

// TestDeleteCommentHandler_RemovesDependentNotification is
// TestDeletePostHandler_RemovesDependentNotifications' counterpart for a
// single comment delete rather than a whole-post delete.
func TestDeleteCommentHandler_RemovesDependentNotification(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	addComment(t, "session-alice", 1, "nice post!")

	var commentID int
	if err := db.QueryRow(`SELECT comment_id FROM notification`).Scan(&commentID); err != nil {
		t.Fatalf("failed to find seeded notification's comment id: %v", err)
	}

	body, _ := json.Marshal(map[string]int{"id": commentID})
	req := httptest.NewRequest(http.MethodPost, "/deleteComment", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-alice"})
	rr := httptest.NewRecorder()

	websocket.DeleteCommentHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notification WHERE comment_id = ?`, commentID).Scan(&count); err != nil {
		t.Fatalf("failed to count notifications: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the comment's notification to be deleted along with it, got %d", count)
	}
}

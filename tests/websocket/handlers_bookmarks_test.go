package websocket_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"sync"
	"testing"
)

func bookmarkPost(t *testing.T, sessionID string, postID int) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	body, _ := json.Marshal(map[string]int{"post_id": postID})
	req := httptest.NewRequest(http.MethodPost, "/bookmarkPost", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	rr := httptest.NewRecorder()
	websocket.BookmarkPostHandler(rr, req)

	var resp struct {
		Bookmarked bool `json:"bookmarked"`
	}
	if rr.Code == http.StatusOK {
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode bookmark response: %v", err)
		}
	}
	return rr, resp.Bookmarked
}

func TestBookmarkPostHandler_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/bookmarkPost", bytes.NewBufferString(`{"post_id":1}`))
	rr := httptest.NewRecorder()

	websocket.BookmarkPostHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestBookmarkPostHandler_RejectsNonexistentPost(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	rr, _ := bookmarkPost(t, "session-alice", 9999)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

func TestBookmarkPostHandler_AddsThenTogglesOffOnSecondRequest(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	rr, bookmarked := bookmarkPost(t, "session-alice", 1)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if !bookmarked {
		t.Fatal("expected bookmarked: true on first request")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bookmark WHERE user_id = 2 AND post_id = 1`).Scan(&count); err != nil {
		t.Fatalf("failed to count bookmark rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 bookmark row, got %d", count)
	}

	rr, bookmarked = bookmarkPost(t, "session-alice", 1)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if bookmarked {
		t.Fatal("expected bookmarked: false on the second (toggle-off) request")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM bookmark WHERE user_id = 2 AND post_id = 1`).Scan(&count); err != nil {
		t.Fatalf("failed to count bookmark rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the bookmark row to be gone after toggling off, got %d", count)
	}
}

// TestBookmarkPostHandler_ConcurrentBookmarkRace_NoServerErrors guards
// toggleBookmark's UNIQUE-constraint backstop — mirrors
// TestReactToPostHandler_ConcurrentInsertRace_NoServerErrors for
// user_post_reaction's identical shape.
func TestBookmarkPostHandler_ConcurrentBookmarkRace_NoServerErrors(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	db.SetMaxOpenConns(1)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	const n = 10
	var wg sync.WaitGroup
	codes := make([]int, n)
	bodies := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body, _ := json.Marshal(map[string]int{"post_id": 1})
			req := httptest.NewRequest(http.MethodPost, "/bookmarkPost", bytes.NewBuffer(body))
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-alice"})
			rr := httptest.NewRecorder()
			websocket.BookmarkPostHandler(rr, req)
			codes[i] = rr.Code
			bodies[i] = rr.Body.String()
		}(i)
	}
	wg.Wait()

	for i, code := range codes {
		if code != http.StatusOK {
			t.Fatalf("request %d: expected status %d, got %d: %s", i, http.StatusOK, code, bodies[i])
		}
	}

	var rowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bookmark WHERE user_id = 2 AND post_id = 1`).Scan(&rowCount); err != nil {
		t.Fatalf("failed to count bookmark rows: %v", err)
	}
	if rowCount > 1 {
		t.Fatalf("expected at most 1 bookmark row for this user+post, got %d — the UNIQUE constraint was violated", rowCount)
	}
}

func TestGetBookmarksHandler_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getBookmarks", nil)
	rr := httptest.NewRecorder()

	websocket.GetBookmarksHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestGetBookmarksHandler_ReturnsOwnBookmarksNewestFirst covers ordering
// (most-recently-bookmarked first, not post-creation order), the
// MyBookmark flag, and that another user's bookmarks never leak in.
func TestGetBookmarksHandler_ReturnsOwnBookmarksNewestFirst(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	// Bookmarked in this order: post 2, then post 1 — the reverse of their
	// own ids/creation order, so "newest first" can only mean
	// newest-bookmarked, not coincidentally matching post id order too.
	if rr, _ := bookmarkPost(t, "session-alice", 2); rr.Code != http.StatusOK {
		t.Fatalf("failed to bookmark post 2: %s", rr.Body.String())
	}
	if rr, _ := bookmarkPost(t, "session-alice", 1); rr.Code != http.StatusOK {
		t.Fatalf("failed to bookmark post 1: %s", rr.Body.String())
	}
	// admin bookmarks post 3 — must never show up in alice's list.
	if rr, _ := bookmarkPost(t, "session-admin", 3); rr.Code != http.StatusOK {
		t.Fatalf("failed to bookmark post 3: %s", rr.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/getBookmarks", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-alice"})
	rr := httptest.NewRecorder()

	websocket.GetBookmarksHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("X-Total-Count"); got != "2" {
		t.Fatalf("expected X-Total-Count 2, got %q", got)
	}

	var posts []websocket.Post
	if err := json.Unmarshal(rr.Body.Bytes(), &posts); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 bookmarked posts, got %d", len(posts))
	}
	if posts[0].PostId != 1 || posts[1].PostId != 2 {
		t.Fatalf("expected newest-bookmarked-first order [1, 2], got [%d, %d]", posts[0].PostId, posts[1].PostId)
	}
	for _, p := range posts {
		if !p.MyBookmark {
			t.Fatalf("expected MyBookmark true for post %d in the bookmarks list itself", p.PostId)
		}
	}
}

// TestDeletePostHandler_RemovesDependentBookmarks guards against the new
// bookmark.post_id foreign key blocking post deletion outright (foreign
// keys are enforced on this connection) once a post has been bookmarked.
func TestDeletePostHandler_RemovesDependentBookmarks(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	if rr, _ := bookmarkPost(t, "session-alice", 1); rr.Code != http.StatusOK {
		t.Fatalf("failed to bookmark post 1: %s", rr.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/deletePost", bytes.NewBufferString(`{"id":1}`))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.DeletePostHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bookmark WHERE post_id = 1`).Scan(&count); err != nil {
		t.Fatalf("failed to count bookmarks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the post's bookmarks to be deleted along with it, got %d", count)
	}
}

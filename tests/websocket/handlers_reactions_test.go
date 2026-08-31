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

type reactionResponse struct {
	LikeCount    int    `json:"like_count"`
	DislikeCount int    `json:"dislike_count"`
	MyReaction   string `json:"my_reaction"`
}

func reactToPost(t *testing.T, sessionID string, postID int, isLiked bool) (*httptest.ResponseRecorder, reactionResponse) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"post_id": postID, "is_liked": isLiked})
	req := httptest.NewRequest(http.MethodPost, "/reactToPost", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	rr := httptest.NewRecorder()
	websocket.ReactToPostHandler(rr, req)

	var resp reactionResponse
	if rr.Code == http.StatusOK {
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode reaction response: %v", err)
		}
	}
	return rr, resp
}

func TestReactToPostHandler_LikeWithNoExistingReaction(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-react", "alice", 2)

	rr, resp := reactToPost(t, "session-react", 1, true)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if resp.LikeCount != 1 || resp.DislikeCount != 0 {
		t.Fatalf("expected like_count=1 dislike_count=0, got %+v", resp)
	}
	if resp.MyReaction != "liked" {
		t.Fatalf("expected my_reaction 'liked', got %q", resp.MyReaction)
	}
}

func TestReactToPostHandler_SameReactionTwiceTogglesOff(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-react", "alice", 2)

	reactToPost(t, "session-react", 1, true)
	rr, resp := reactToPost(t, "session-react", 1, true)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if resp.LikeCount != 0 || resp.DislikeCount != 0 {
		t.Fatalf("expected both counts back to 0 after toggling off, got %+v", resp)
	}
	if resp.MyReaction != "none" {
		t.Fatalf("expected my_reaction 'none' after toggling off, got %q", resp.MyReaction)
	}
}

func TestReactToPostHandler_SwitchingFromLikeToDislike(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-react", "alice", 2)

	reactToPost(t, "session-react", 1, true)
	rr, resp := reactToPost(t, "session-react", 1, false)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if resp.LikeCount != 0 || resp.DislikeCount != 1 {
		t.Fatalf("expected like_count=0 dislike_count=1 after switching, got %+v", resp)
	}
	if resp.MyReaction != "disliked" {
		t.Fatalf("expected my_reaction 'disliked', got %q", resp.MyReaction)
	}
}

func TestReactToPostHandler_CountsAreSharedAcrossUsers(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	reactToPost(t, "session-alice", 1, true)
	rr, resp := reactToPost(t, "session-actual", 1, true)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if resp.LikeCount != 2 {
		t.Fatalf("expected like_count=2 with two different users liking, got %+v", resp)
	}
	if resp.MyReaction != "liked" {
		t.Fatalf("expected the responding user's own reaction to be 'liked', got %q", resp.MyReaction)
	}
}

func TestReactToPostHandler_RejectsNonexistentPost(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-react", "alice", 2)

	rr, _ := reactToPost(t, "session-react", 99999, true)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

func TestReactToPostHandler_RejectsUnauthenticated(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body, _ := json.Marshal(map[string]any{"post_id": 1, "is_liked": true})
	req := httptest.NewRequest(http.MethodPost, "/reactToPost", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	websocket.ReactToPostHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}
}

func TestReactToPostHandler_RejectsInvalidPostID(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-react", "alice", 2)

	rr, _ := reactToPost(t, "session-react", 0, true)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestAllPostsHandler_IncludesReactionDataForAnonymousViewer(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	reactToPost(t, "session-alice", 1, true)

	req := httptest.NewRequest(http.MethodGet, "/getAllPosts", nil)
	rr := httptest.NewRecorder()
	websocket.AllPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	var post1 *websocket.Post
	for i := range posts {
		if posts[i].PostId == 1 {
			post1 = &posts[i]
		}
	}
	if post1 == nil {
		t.Fatal("expected seeded post 1 in the response")
	}
	if post1.LikeCount != 1 {
		t.Fatalf("expected post 1 to show like_count=1, got %d", post1.LikeCount)
	}
	// An anonymous (unauthenticated) request never gets a personal
	// reaction attached, even though the aggregate counts are still public.
	if post1.MyReaction != "none" {
		t.Fatalf("expected an anonymous viewer's MyReaction to be 'none', got %q", post1.MyReaction)
	}
}

func TestGetPostHandler_IncludesViewersOwnReaction(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	reactToPost(t, "session-alice", 1, false)

	req := httptest.NewRequest(http.MethodGet, "/getPost?id=1", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-alice"})
	rr := httptest.NewRecorder()
	websocket.GetPostHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var post websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&post); err != nil {
		t.Fatalf("failed to decode post: %v", err)
	}
	if post.DislikeCount != 1 {
		t.Fatalf("expected dislike_count=1, got %d", post.DislikeCount)
	}
	if post.MyReaction != "disliked" {
		t.Fatalf("expected the authenticated viewer's own reaction to be 'disliked', got %q", post.MyReaction)
	}
}

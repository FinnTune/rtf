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

func postIDOrder(t *testing.T, posts []websocket.Post) []int {
	t.Helper()
	ids := make([]int, len(posts))
	for i, p := range posts {
		ids[i] = p.PostId
	}
	return ids
}

func TestAllPostsHandler_SortMostLiked(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	// Seeded posts are 1 (actual_user), 2 (admin), 3 (admin), all created at
	// the same instant, so newest order alone would rank them 3, 2, 1. Give
	// post 1 two likes so most_liked should rank it first despite that.
	reactToPost(t, "session-alice", 1, true)
	reactToPost(t, "session-actual", 1, true)

	req := httptest.NewRequest(http.MethodGet, "/getAllPosts?sort=most_liked", nil)
	rr := httptest.NewRecorder()
	websocket.AllPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) == 0 || posts[0].PostId != 1 {
		t.Fatalf("expected post 1 (most liked) first, got order %v", postIDOrder(t, posts))
	}
	if posts[0].LikeCount != 2 {
		t.Fatalf("expected post 1 to show like_count=2, got %d", posts[0].LikeCount)
	}
}

func TestAllPostsHandler_SortMostCommented(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// Seeded post 1 already has one comment (post 2 and 3 have none), so
	// most_commented should rank it first even though it's not the newest.
	req := httptest.NewRequest(http.MethodGet, "/getAllPosts?sort=most_commented", nil)
	rr := httptest.NewRecorder()
	websocket.AllPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) == 0 || posts[0].PostId != 1 {
		t.Fatalf("expected post 1 (most commented) first, got order %v", postIDOrder(t, posts))
	}
}

func TestAllPostsHandler_DefaultSortIsNewest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

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
	// Seeded posts share the same created_at, so id DESC breaks the tie:
	// newest-first means highest id (3) comes first.
	if len(posts) == 0 || posts[0].PostId != 3 {
		t.Fatalf("expected post 3 (highest id, tie-broken) first by default, got order %v", postIDOrder(t, posts))
	}
}

func TestAllPostsHandler_RejectsInvalidSort(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getAllPosts?sort=bogus", nil)
	rr := httptest.NewRecorder()
	websocket.AllPostsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestPostsByCategoryHandler_SortMostLiked(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	// Posts 2 and 3 are both in the 'Cuisine' category (seeded). Post 3 is
	// created after post 2 in insertion order but they share a timestamp;
	// give post 2 a like so most_liked should surface it first.
	reactToPost(t, "session-alice", 2, true)

	body := `{"categories":["Cuisine"]}`
	req := httptest.NewRequest(http.MethodPost, "/postsByCategory?sort=most_liked", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	websocket.PostsByCategoryHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) == 0 || posts[0].PostId != 2 {
		t.Fatalf("expected post 2 (most liked) first, got order %v", postIDOrder(t, posts))
	}
}

func TestGetPostsByAuthorHandler_SortMostLiked(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	// Posts 2 and 3 are both authored by 'admin'. Like post 2 so
	// most_liked should surface it first despite post 3 having a higher id.
	reactToPost(t, "session-alice", 2, true)

	req := httptest.NewRequest(http.MethodGet, "/getPostsByAuthor?author=admin&sort=most_liked", nil)
	rr := httptest.NewRecorder()
	websocket.GetPostsByAuthorHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) == 0 || posts[0].PostId != 2 {
		t.Fatalf("expected post 2 (most liked) first, got order %v", postIDOrder(t, posts))
	}
}

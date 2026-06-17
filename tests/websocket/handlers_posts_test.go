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

func TestAllPostsHandler_ReturnsPosts(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getAllPosts", nil)
	rr := httptest.NewRecorder()

	websocket.AllPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(posts))
	}
}

func TestGetCommentsHandler_ReturnsComments(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/comments?postId=1", nil)
	rr := httptest.NewRecorder()

	websocket.GetCommentsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var comments []websocket.Comment
	if err := json.NewDecoder(rr.Body).Decode(&comments); err != nil {
		t.Fatalf("failed to decode comments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Content != "existing comment" {
		t.Fatalf("unexpected comment content: %q", comments[0].Content)
	}
	if comments[0].Username != "actual_user" {
		t.Fatalf("expected author actual_user, got %q", comments[0].Username)
	}
}

func TestGetCommentsHandler_MissingPostId(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/comments", nil)
	rr := httptest.NewRecorder()

	websocket.GetCommentsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetCommentsHandler_InvalidPostId(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/comments?postId=abc", nil)
	rr := httptest.NewRecorder()

	websocket.GetCommentsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetCommentsHandler_RejectsNonGetMethod(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/comments?postId=1", nil)
	rr := httptest.NewRecorder()

	websocket.GetCommentsHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestPostsByCategoryHandler_FiltersByCategory(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"categories":["Cuisine"]}`
	req := httptest.NewRequest(http.MethodPost, "/getPostsByCategory", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.PostsByCategoryHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 cuisine posts, got %d", len(posts))
	}

	titles := map[string]bool{}
	for _, post := range posts {
		titles[post.Title] = true
	}
	if !titles["Asian Food"] || !titles["Best Sushi"] {
		t.Fatalf("unexpected posts returned: %+v", posts)
	}
}

func TestPostsByCategoryHandler_InvalidJSON(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/getPostsByCategory", bytes.NewBufferString(`{bad`))
	rr := httptest.NewRecorder()

	websocket.PostsByCategoryHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestAddPost_StoresCategoryRelations(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	websocket.AddAuthenticatedClient("session-cat", "actual_user", 42)

	payload := map[string]any{
		"title":   "Categorized",
		"content": "With categories",
		"categories": []map[string]any{
			{"id": 1, "name": "Cuisine"},
			{"id": 5, "name": "Code"},
		},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/addPost", bytes.NewBuffer(data))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-cat"})
	rr := httptest.NewRecorder()

	websocket.AddPost(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var postID int64
	err := db.QueryRow(`SELECT id FROM post WHERE title = ?`, "Categorized").Scan(&postID)
	if err != nil {
		t.Fatalf("failed to find new post: %v", err)
	}

	var relationCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM category_relation WHERE post_id = ?`, postID).Scan(&relationCount)
	if err != nil {
		t.Fatalf("failed to count category relations: %v", err)
	}
	if relationCount != 2 {
		t.Fatalf("expected 2 category relations, got %d", relationCount)
	}
}
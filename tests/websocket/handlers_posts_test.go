package websocket_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"strings"
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
	if got := rr.Header().Get("X-Total-Count"); got != "3" {
		t.Fatalf("expected X-Total-Count 3, got %q", got)
	}
}

func TestAllPostsHandler_RespectsLimitAndOffset(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getAllPosts?limit=2&offset=0", nil)
	rr := httptest.NewRecorder()
	websocket.AllPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var page1 []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&page1); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("expected 2 posts on first page, got %d", len(page1))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/getAllPosts?limit=2&offset=2", nil)
	rr2 := httptest.NewRecorder()
	websocket.AllPostsHandler(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr2.Code)
	}
	var page2 []websocket.Post
	if err := json.NewDecoder(rr2.Body).Decode(&page2); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("expected 1 post on second page, got %d", len(page2))
	}
	if page1[0].PostId == page2[0].PostId {
		t.Fatalf("pages overlap: post %d returned on both pages", page1[0].PostId)
	}
}

func TestAllPostsHandler_InvalidParamsFallBackToDefaults(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getAllPosts?limit=abc&offset=-5", nil)
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
		t.Fatalf("expected default limit/offset to return all 3 seed posts, got %d", len(posts))
	}
}

func TestAllPostsHandler_CapsLimitAtMax(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	for i := 0; i < 60; i++ {
		_, err := db.Exec(
			`INSERT INTO post (user_id, title, content, author, created_at) VALUES (1, ?, 'bulk content', 'admin', datetime('now'))`,
			fmt.Sprintf("Bulk Post %d", i),
		)
		if err != nil {
			t.Fatalf("failed to seed bulk post: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/getAllPosts?limit=9999", nil)
	rr := httptest.NewRecorder()
	websocket.AllPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) != 50 {
		t.Fatalf("expected limit to be capped at 50, got %d posts", len(posts))
	}
	if got := rr.Header().Get("X-Total-Count"); got != "63" {
		t.Fatalf("expected X-Total-Count 63, got %q", got)
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
	if got := rr.Header().Get("X-Total-Count"); got != "1" {
		t.Fatalf("expected X-Total-Count 1, got %q", got)
	}
}

func TestGetCommentsHandler_RespectsLimitAndOffset(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	// Seed data already has 1 comment on post 1; add more so pagination is
	// actually exercised, with distinct, ordered created_at values so the
	// ORDER BY is unambiguous.
	for i := 0; i < 5; i++ {
		_, err := db.Exec(
			`INSERT INTO comment (user_id, post_id, content, created_at) VALUES (42, 1, ?, datetime('now', ?))`,
			fmt.Sprintf("bulk comment %d", i), fmt.Sprintf("+%d seconds", i+1),
		)
		if err != nil {
			t.Fatalf("failed to seed bulk comment: %v", err)
		}
	}
	// 6 comments total on post 1 now.

	req := httptest.NewRequest(http.MethodGet, "/comments?postId=1&limit=2&offset=0", nil)
	rr := httptest.NewRecorder()
	websocket.GetCommentsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var page1 []websocket.Comment
	if err := json.NewDecoder(rr.Body).Decode(&page1); err != nil {
		t.Fatalf("failed to decode comments: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("expected 2 comments on first page, got %d", len(page1))
	}
	if got := rr.Header().Get("X-Total-Count"); got != "6" {
		t.Fatalf("expected X-Total-Count 6, got %q", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/comments?postId=1&limit=2&offset=2", nil)
	rr2 := httptest.NewRecorder()
	websocket.GetCommentsHandler(rr2, req2)

	var page2 []websocket.Comment
	if err := json.NewDecoder(rr2.Body).Decode(&page2); err != nil {
		t.Fatalf("failed to decode comments: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("expected 2 comments on second page, got %d", len(page2))
	}
	if page1[0].ID == page2[0].ID {
		t.Fatalf("pages overlap: comment %d returned on both pages", page1[0].ID)
	}
}

func TestGetCommentsHandler_CapsLimitAtMax(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	for i := 0; i < 110; i++ {
		_, err := db.Exec(
			`INSERT INTO comment (user_id, post_id, content, created_at) VALUES (42, 1, ?, datetime('now'))`,
			fmt.Sprintf("bulk comment %d", i),
		)
		if err != nil {
			t.Fatalf("failed to seed bulk comment: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/comments?postId=1&limit=9999", nil)
	rr := httptest.NewRecorder()
	websocket.GetCommentsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var comments []websocket.Comment
	if err := json.NewDecoder(rr.Body).Decode(&comments); err != nil {
		t.Fatalf("failed to decode comments: %v", err)
	}
	if len(comments) != 100 {
		t.Fatalf("expected limit to be capped at 100, got %d comments", len(comments))
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
	if got := rr.Header().Get("X-Total-Count"); got != "2" {
		t.Fatalf("expected X-Total-Count 2, got %q", got)
	}
}

func TestPostsByCategoryHandler_RespectsLimitAndOffset(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"categories":["Cuisine"]}`

	req1 := httptest.NewRequest(http.MethodPost, "/getPostsByCategory?limit=1&offset=0", bytes.NewBufferString(body))
	rr1 := httptest.NewRecorder()
	websocket.PostsByCategoryHandler(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr1.Code, rr1.Body.String())
	}
	var page1 []websocket.Post
	if err := json.NewDecoder(rr1.Body).Decode(&page1); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(page1) != 1 {
		t.Fatalf("expected 1 post on first page, got %d", len(page1))
	}
	if got := rr1.Header().Get("X-Total-Count"); got != "2" {
		t.Fatalf("expected X-Total-Count 2, got %q", got)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/getPostsByCategory?limit=1&offset=1", bytes.NewBufferString(body))
	rr2 := httptest.NewRecorder()
	websocket.PostsByCategoryHandler(rr2, req2)

	var page2 []websocket.Post
	if err := json.NewDecoder(rr2.Body).Decode(&page2); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("expected 1 post on second page, got %d", len(page2))
	}
	if page1[0].PostId == page2[0].PostId {
		t.Fatalf("pages overlap: post %d returned on both pages", page1[0].PostId)
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

func TestSearchPostsHandler_MatchesTitle(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/searchPosts?q=Asian", nil)
	rr := httptest.NewRecorder()

	websocket.SearchPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) != 1 || posts[0].Title != "Asian Food" {
		t.Fatalf("expected to find 'Asian Food', got %+v", posts)
	}
}

func TestSearchPostsHandler_MatchesContentCaseInsensitive(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	// "Thai Khun Mom" is in the content of post 2, not its title. SQLite's
	// LIKE is case-insensitive for ASCII by default.
	req := httptest.NewRequest(http.MethodGet, "/searchPosts?q=thai", nil)
	rr := httptest.NewRecorder()

	websocket.SearchPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) != 1 || posts[0].Title != "Asian Food" {
		t.Fatalf("expected content match for 'Asian Food', got %+v", posts)
	}
}

func TestSearchPostsHandler_NoMatches(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/searchPosts?q=nonexistentxyz", nil)
	rr := httptest.NewRecorder()

	websocket.SearchPostsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected no matches, got %+v", posts)
	}
}

func TestSearchPostsHandler_RejectsEmptyQuery(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/searchPosts?q=", nil)
	rr := httptest.NewRecorder()

	websocket.SearchPostsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestSearchPostsHandler_RejectsOverlongQuery(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	longQuery := strings.Repeat("a", 101)
	req := httptest.NewRequest(http.MethodGet, "/searchPosts?q="+longQuery, nil)
	rr := httptest.NewRecorder()

	websocket.SearchPostsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestSearchPostsHandler_RejectsNonGetMethod(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/searchPosts?q=Asian", nil)
	rr := httptest.NewRecorder()

	websocket.SearchPostsHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestGetPostHandler_ReturnsPost(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getPost?id=2", nil)
	rr := httptest.NewRecorder()

	websocket.GetPostHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var post websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&post); err != nil {
		t.Fatalf("failed to decode post: %v", err)
	}
	if post.PostId != 2 || post.Title != "Asian Food" {
		t.Fatalf("unexpected post: %+v", post)
	}
}

func TestGetPostHandler_NotFound(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getPost?id=9999", nil)
	rr := httptest.NewRecorder()

	websocket.GetPostHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestGetPostHandler_RejectsInvalidId(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getPost?id=abc", nil)
	rr := httptest.NewRecorder()

	websocket.GetPostHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetPostHandler_RejectsNonGetMethod(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/getPost?id=2", nil)
	rr := httptest.NewRecorder()

	websocket.GetPostHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestGetCategoriesHandler_ReturnsSeededCategories(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getCategories", nil)
	rr := httptest.NewRecorder()

	websocket.GetCategoriesHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var categories []websocket.Category
	if err := json.NewDecoder(rr.Body).Decode(&categories); err != nil {
		t.Fatalf("failed to decode categories: %v", err)
	}
	// Seed data (testutil.SetupForumDB) has 3 categories: Cuisine, Places, Code.
	if len(categories) != 3 {
		t.Fatalf("expected 3 categories, got %d: %+v", len(categories), categories)
	}
	names := map[string]bool{}
	for _, c := range categories {
		names[c.Name] = true
		if c.ID == 0 {
			t.Fatalf("expected a non-zero category id, got %+v", c)
		}
	}
	if !names["Cuisine"] || !names["Places"] || !names["Code"] {
		t.Fatalf("unexpected category set: %+v", categories)
	}
}

func TestGetCategoriesHandler_RejectsNonGetMethod(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/getCategories", nil)
	rr := httptest.NewRecorder()

	websocket.GetCategoriesHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}
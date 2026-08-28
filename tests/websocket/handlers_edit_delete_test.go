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

func TestEditPostHandler_UpdatesOwnPost(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	body := `{"id":1,"title":"Updated Title","content":"Updated content"}`
	req := httptest.NewRequest(http.MethodPost, "/editPost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.EditPostHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var title, content string
	if err := db.QueryRow(`SELECT title, content FROM post WHERE id = 1`).Scan(&title, &content); err != nil {
		t.Fatalf("failed to query updated post: %v", err)
	}
	if title != "Updated Title" || content != "Updated content" {
		t.Fatalf("post not updated: title=%q content=%q", title, content)
	}
}

func TestEditPostHandler_RejectsNonOwner(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	// post id 2 belongs to user_id 1 (admin), not 42
	body := `{"id":2,"title":"Hijacked","content":"Hijacked content"}`
	req := httptest.NewRequest(http.MethodPost, "/editPost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.EditPostHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}

	var title string
	if err := db.QueryRow(`SELECT title FROM post WHERE id = 2`).Scan(&title); err != nil {
		t.Fatalf("failed to query post: %v", err)
	}
	if title == "Hijacked" {
		t.Fatalf("post was updated despite ownership check")
	}
}

func TestEditPostHandler_RejectsUnauthenticated(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"id":1,"title":"Updated Title","content":"Updated content"}`
	req := httptest.NewRequest(http.MethodPost, "/editPost", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.EditPostHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestEditPostHandler_RejectsInvalidContent(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	body := `{"id":1,"title":"","content":"Updated content"}`
	req := httptest.NewRequest(http.MethodPost, "/editPost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.EditPostHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestEditPostHandler_NotFound(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	body := `{"id":9999,"title":"Title","content":"Content"}`
	req := httptest.NewRequest(http.MethodPost, "/editPost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.EditPostHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestEditPostHandler_ReplacesCategoryRelations(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	// Seed data: post 1 is tagged with category 5 (Code) only.
	var before int
	db.QueryRow(`SELECT COUNT(*) FROM category_relation WHERE post_id = 1 AND category_id = 5`).Scan(&before)
	if before != 1 {
		t.Fatalf("expected seed post 1 to start tagged with category 5, got count %d", before)
	}

	body := `{"id":1,"title":"Updated Title","content":"Updated content","categories":[{"id":1,"name":"Cuisine"}]}`
	req := httptest.NewRequest(http.MethodPost, "/editPost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.EditPostHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var codeCount, cuisineCount int
	db.QueryRow(`SELECT COUNT(*) FROM category_relation WHERE post_id = 1 AND category_id = 5`).Scan(&codeCount)
	db.QueryRow(`SELECT COUNT(*) FROM category_relation WHERE post_id = 1 AND category_id = 1`).Scan(&cuisineCount)
	if codeCount != 0 {
		t.Fatalf("expected old category relation (Code) to be removed, found %d", codeCount)
	}
	if cuisineCount != 1 {
		t.Fatalf("expected new category relation (Cuisine) to be added, found %d", cuisineCount)
	}
}

func TestEditPostHandler_RejectsTooManyCategories(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	cats := make([]string, 21)
	for i := range cats {
		cats[i] = fmt.Sprintf(`{"id":%d,"name":"c%d"}`, i+1, i+1)
	}
	body := fmt.Sprintf(`{"id":1,"title":"Title","content":"Content","categories":[%s]}`, strings.Join(cats, ","))
	req := httptest.NewRequest(http.MethodPost, "/editPost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.EditPostHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetPostCategoriesHandler_ReturnsAssignedCategories(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getPostCategories?postId=1", nil)
	rr := httptest.NewRecorder()

	websocket.GetPostCategoriesHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var categories []websocket.Category
	if err := json.NewDecoder(rr.Body).Decode(&categories); err != nil {
		t.Fatalf("failed to decode categories: %v", err)
	}
	if len(categories) != 1 || categories[0].Name != "Code" {
		t.Fatalf("expected post 1 to be tagged with only Code, got %+v", categories)
	}
}

func TestGetPostCategoriesHandler_RejectsInvalidPostId(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/getPostCategories?postId=abc", nil)
	rr := httptest.NewRecorder()

	websocket.GetPostCategoriesHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetPostCategoriesHandler_RejectsNonGetMethod(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/getPostCategories?postId=1", nil)
	rr := httptest.NewRecorder()

	websocket.GetPostCategoriesHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestDeletePostHandler_DeletesOwnPostAndDependents(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	// post id 1 owns comment id 1 and category_relation id 3 in the seed data.
	body := `{"id":1}`
	req := httptest.NewRequest(http.MethodPost, "/deletePost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.DeletePostHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var postCount, commentCount, relationCount int
	db.QueryRow(`SELECT COUNT(*) FROM post WHERE id = 1`).Scan(&postCount)
	db.QueryRow(`SELECT COUNT(*) FROM comment WHERE post_id = 1`).Scan(&commentCount)
	db.QueryRow(`SELECT COUNT(*) FROM category_relation WHERE post_id = 1`).Scan(&relationCount)

	if postCount != 0 {
		t.Fatalf("expected post to be deleted, found %d", postCount)
	}
	if commentCount != 0 {
		t.Fatalf("expected post's comments to be deleted, found %d", commentCount)
	}
	if relationCount != 0 {
		t.Fatalf("expected post's category relations to be deleted, found %d", relationCount)
	}
}

func TestDeletePostHandler_RejectsNonOwner(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	body := `{"id":2}`
	req := httptest.NewRequest(http.MethodPost, "/deletePost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.DeletePostHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}

	var postCount int
	db.QueryRow(`SELECT COUNT(*) FROM post WHERE id = 2`).Scan(&postCount)
	if postCount != 1 {
		t.Fatalf("post should not have been deleted")
	}
}

func TestEditCommentHandler_UpdatesOwnComment(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	body := `{"id":1,"content":"edited comment"}`
	req := httptest.NewRequest(http.MethodPost, "/editComment", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.EditCommentHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var content string
	if err := db.QueryRow(`SELECT content FROM comment WHERE id = 1`).Scan(&content); err != nil {
		t.Fatalf("failed to query comment: %v", err)
	}
	if content != "edited comment" {
		t.Fatalf("comment not updated: got %q", content)
	}
}

func TestEditCommentHandler_RejectsNonOwner(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	// alice (user_id 2) does not own comment id 1 (owned by user_id 42)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	body := `{"id":1,"content":"hijacked"}`
	req := httptest.NewRequest(http.MethodPost, "/editComment", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-alice"})
	rr := httptest.NewRecorder()

	websocket.EditCommentHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}

	var content string
	db.QueryRow(`SELECT content FROM comment WHERE id = 1`).Scan(&content)
	if content == "hijacked" {
		t.Fatalf("comment was updated despite ownership check")
	}
}

func TestDeleteCommentHandler_DeletesOwnComment(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	body := `{"id":1}`
	req := httptest.NewRequest(http.MethodPost, "/deleteComment", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.DeleteCommentHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM comment WHERE id = 1`).Scan(&count)
	if count != 0 {
		t.Fatalf("expected comment to be deleted, found %d", count)
	}
}

func TestDeleteCommentHandler_RejectsNonOwner(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	body := `{"id":1}`
	req := httptest.NewRequest(http.MethodPost, "/deleteComment", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-alice"})
	rr := httptest.NewRecorder()

	websocket.DeleteCommentHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM comment WHERE id = 1`).Scan(&count)
	if count != 1 {
		t.Fatalf("comment should not have been deleted")
	}
}

func TestEditPostHandler_RejectsNonPostMethod(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/editPost", nil)
	rr := httptest.NewRecorder()

	websocket.EditPostHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestDeleteCommentHandler_InvalidJSON(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	req := httptest.NewRequest(http.MethodPost, "/deleteComment", bytes.NewBufferString(`{bad`))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	rr := httptest.NewRecorder()

	websocket.DeleteCommentHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

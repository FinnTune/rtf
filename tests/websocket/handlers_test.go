package websocket_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"testing"
)

func TestCheckOrigin_DefaultAndEnvOverride(t *testing.T) {
	_ = os.Unsetenv("ALLOWED_ORIGIN")
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://localhost:8443")
	if !websocket.CheckOriginForTest(req) {
		t.Fatalf("expected default origin to be allowed")
	}

	t.Setenv("ALLOWED_ORIGIN", "https://example.com")
	req2 := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req2.Header.Set("Origin", "https://example.com")
	if !websocket.CheckOriginForTest(req2) {
		t.Fatalf("expected env-configured origin to be allowed")
	}
}

func TestCheckOrigin_RejectsWrongOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGIN", "https://example.com")
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	if websocket.CheckOriginForTest(req) {
		t.Fatal("expected disallowed origin to be rejected")
	}
}

func TestAddPost_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"title":"t","content":"c","categories":[{"id":1,"name":"Code"}]}`
	req := httptest.NewRequest(http.MethodPost, "/addPost", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.AddPost(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAddPost_UsesAuthenticatedSessionIdentity(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	websocket.AddAuthenticatedClient("session-123", "actual_user", 42)

	payload := map[string]any{
		"title":   "Hello",
		"content": "World",
		"author":  "spoofed_user",
		"userID":  999,
		"categories": []map[string]any{
			{"id": 1, "name": "Code"},
		},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/addPost", bytes.NewBuffer(data))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
	rr := httptest.NewRecorder()

	websocket.AddPost(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var userID int
	var author string
	err := db.QueryRow(`SELECT user_id, author FROM post WHERE title = ?`, "Hello").Scan(&userID, &author)
	if err != nil {
		t.Fatalf("failed to fetch inserted post: %v", err)
	}
	if userID != 42 || author != "actual_user" {
		t.Fatalf("expected session identity (42, actual_user), got (%d, %s)", userID, author)
	}
}

func TestAddComment_UsesAuthenticatedSessionIdentity(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)

	websocket.AddAuthenticatedClient("session-abc", "actual_user", 42)

	body := `{"post_id":1,"content":"comment body","user_id":999}`
	req := httptest.NewRequest(http.MethodPost, "/addcomment", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-abc"})
	rr := httptest.NewRecorder()

	websocket.AddCommentHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var userID int
	var content string
	err := db.QueryRow(`SELECT user_id, content FROM comment ORDER BY id DESC LIMIT 1`).Scan(&userID, &content)
	if err != nil {
		t.Fatalf("failed to fetch inserted comment: %v", err)
	}
	if userID != 42 {
		t.Fatalf("expected authenticated user id 42, got %d", userID)
	}
	if content != "comment body" {
		t.Fatalf("expected content 'comment body', got %q", content)
	}
}

func TestAddComment_RejectsUnauthenticatedRequest(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"post_id":1,"content":"comment body"}`
	req := httptest.NewRequest(http.MethodPost, "/addcomment", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.AddCommentHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAddComment_RejectsNonPostMethod(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/addcomment", nil)
	rr := httptest.NewRecorder()

	websocket.AddCommentHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

package websocket

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"rtForum/database"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	schema := `
	PRAGMA foreign_keys = ON;
	CREATE TABLE user (
		id INTEGER NOT NULL PRIMARY KEY,
		uname VARCHAR(30) NOT NULL
	);
	CREATE TABLE post (
		id INTEGER NOT NULL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		title VARCHAR(30) NOT NULL,
		content VARCHAR(150) NOT NULL,
		author VARCHAR(30) NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY(user_id) REFERENCES user(id)
	);
	CREATE TABLE category_relation (
		id INTEGER NOT NULL PRIMARY KEY,
		category_id INTEGER NOT NULL,
		post_id INTEGER NOT NULL
	);
	CREATE TABLE comment (
		id INTEGER NOT NULL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		post_id INTEGER NOT NULL,
		content VARCHAR(150) NOT NULL,
		created_at DATETIME NOT NULL
	);
	`

	if _, err = db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	if _, err = db.Exec(`INSERT INTO user (id, uname) VALUES (42, 'actual_user');`); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if _, err = db.Exec(`INSERT INTO post (id, user_id, title, content, author, created_at) VALUES (1, 42, 'seed', 'seed', 'actual_user', datetime('now'));`); err != nil {
		t.Fatalf("failed to seed post: %v", err)
	}

	return db
}

func resetTestManager() {
	manager = newManager(context.Background())
}

func addAuthenticatedClient(sessionID, username string, userID int) {
	client := &Client{
		sessionID: sessionID,
		loggedIn:  true,
		username:  username,
		userID:    userID,
	}
	manager.clients[client] = true
}

func TestCheckOrigin_DefaultAndEnvOverride(t *testing.T) {
	_ = os.Unsetenv("ALLOWED_ORIGIN")
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://localhost:8443")
	if !checkOrigin(req) {
		t.Fatalf("expected default origin to be allowed")
	}

	t.Setenv("ALLOWED_ORIGIN", "https://example.com")
	req2 := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req2.Header.Set("Origin", "https://example.com")
	if !checkOrigin(req2) {
		t.Fatalf("expected env-configured origin to be allowed")
	}
}

func TestAddPost_RejectsUnauthenticatedRequest(t *testing.T) {
	resetTestManager()
	db := setupTestDB(t)
	defer db.Close()
	database.ForumDB = db

	body := `{"title":"t","content":"c","categories":[{"id":1,"name":"Code"}]}`
	req := httptest.NewRequest(http.MethodPost, "/addPost", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	AddPost(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAddPost_UsesAuthenticatedSessionIdentity(t *testing.T) {
	resetTestManager()
	db := setupTestDB(t)
	defer db.Close()
	database.ForumDB = db

	addAuthenticatedClient("session-123", "actual_user", 42)

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

	AddPost(rr, req)

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
	resetTestManager()
	db := setupTestDB(t)
	defer db.Close()
	database.ForumDB = db

	addAuthenticatedClient("session-abc", "actual_user", 42)

	body := `{"post_id":1,"content":"comment body","user_id":999}`
	req := httptest.NewRequest(http.MethodPost, "/addcomment", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-abc"})
	rr := httptest.NewRecorder()

	AddCommentHandler(rr, req)

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
	resetTestManager()
	db := setupTestDB(t)
	defer db.Close()
	database.ForumDB = db

	body := `{"post_id":1,"content":"comment body"}`
	req := httptest.NewRequest(http.MethodPost, "/addcomment", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	AddCommentHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

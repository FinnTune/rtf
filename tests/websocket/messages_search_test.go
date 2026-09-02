package websocket_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"testing"
)

func searchMessagesRequest(sessionID, query string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/searchMessages?q="+query, nil)
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	}
	return req
}

func TestSearchMessagesHandler_MatchesContent(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	rr := httptest.NewRecorder()
	websocket.SearchMessagesHandler(rr, searchMessagesRequest("session-admin", "hello"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var results []websocket.ChatHistoryMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to decode search results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Message != "hello alice" || results[0].From != "admin" {
		t.Fatalf("unexpected result: %+v", results[0])
	}
}

func TestSearchMessagesHandler_CaseInsensitive(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	rr := httptest.NewRecorder()
	websocket.SearchMessagesHandler(rr, searchMessagesRequest("session-admin", "HELLO"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var results []websocket.ChatHistoryMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to decode search results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected a case-insensitive match, got %d results", len(results))
	}
}

func TestSearchMessagesHandler_OnlySearchesOwnConversations(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	// actual_user (42) is not a member of the seeded admin<->alice
	// conversation that contains "hello alice".
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	rr := httptest.NewRecorder()
	websocket.SearchMessagesHandler(rr, searchMessagesRequest("session-actual", "hello"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var results []websocket.ChatHistoryMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to decode search results: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results for a conversation the requester isn't part of, got %d: %+v", len(results), results)
	}
}

func TestSearchMessagesHandler_NoMatches(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	rr := httptest.NewRecorder()
	websocket.SearchMessagesHandler(rr, searchMessagesRequest("session-admin", "nonexistentword"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	var results []websocket.ChatHistoryMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to decode search results: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results, got %d", len(results))
	}
}

func TestSearchMessagesHandler_RejectsUnauthenticated(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	rr := httptest.NewRecorder()
	websocket.SearchMessagesHandler(rr, searchMessagesRequest("", "hello"))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}
}

func TestSearchMessagesHandler_RejectsEmptyQuery(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	rr := httptest.NewRecorder()
	websocket.SearchMessagesHandler(rr, searchMessagesRequest("session-admin", ""))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestSearchMessagesHandler_RejectsNonGetMethod(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	req := httptest.NewRequest(http.MethodPost, "/searchMessages?q=hello", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()
	websocket.SearchMessagesHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d: %s", http.StatusMethodNotAllowed, rr.Code, rr.Body.String())
	}
}

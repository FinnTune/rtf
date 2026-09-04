package websocket_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"testing"
)

func TestCSRFProtect_RejectsMismatchedOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGIN", "https://example.com")
	called := false
	handler := websocket.CSRFProtect(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
	if called {
		t.Fatal("expected wrapped handler not to run when origin check fails")
	}
}

func TestCSRFProtect_AllowsMatchingOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGIN", "https://example.com")
	called := false
	handler := websocket.CSRFProtect(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !called {
		t.Fatal("expected wrapped handler to run when origin check passes")
	}
}

func TestCSRFProtect_RejectsMissingOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGIN", "https://example.com")
	called := false
	handler := websocket.CSRFProtect(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
	if called {
		t.Fatal("expected wrapped handler not to run when Origin header is absent")
	}
}

// End-to-end check that CSRFProtect is actually effective in front of a real
// mutating handler, not just in isolation with a stub - a routing change
// that dropped or reordered the wrapping (e.g. RequireAdmin(CSRFProtect(...))
// vs CSRFProtect(RequireAdmin(...))) would otherwise go uncaught.
func TestCSRFProtect_ProtectsRealHandlerFromForgedOrigin(t *testing.T) {
	_ = os.Unsetenv("ALLOWED_ORIGIN") // falls back to the default https://localhost:8443
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-owner", "actual_user", 42)

	protected := websocket.CSRFProtect(websocket.AddPost)

	body := `{"title":"Forged","content":"Forged via CSRF"}`
	req := httptest.NewRequest(http.MethodPost, "/addPost", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-owner"})
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()

	protected(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM post WHERE title = 'Forged'`).Scan(&count)
	if count != 0 {
		t.Fatal("expected forged-origin request not to reach AddPost and create a post")
	}
}

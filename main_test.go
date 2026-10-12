package main

import (
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"testing"
)

func TestHealthzHandler_ReturnsOKWhenDatabaseReachable(t *testing.T) {
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	healthzHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHealthzHandler_ReturnsServiceUnavailableWhenDatabaseUnreachable(t *testing.T) {
	db := testutil.UseForumDB(t)
	db.Close()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	healthzHandler(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
}

func TestHealthzHandler_RejectsNonGetMethod(t *testing.T) {
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rr := httptest.NewRecorder()

	healthzHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHealthzHandler_AllowsHeadMethod(t *testing.T) {
	testutil.UseForumDB(t)

	req := httptest.NewRequest(http.MethodHead, "/healthz", nil)
	rr := httptest.NewRecorder()

	healthzHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestSecurityHeaders_SetsExpectedHeaders guards the response headers every
// request gets wrapped with, including Permissions-Policy denying every
// sensitive feature outright (see securityHeaders' doc comment on why: the
// frontend has no legitimate use for any of them today).
func TestSecurityHeaders_SetsExpectedHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=(), usb=(), payment=(), accelerometer=(), gyroscope=(), magnetometer=()",
	} {
		if got := rr.Header().Get(header); got != want {
			t.Errorf("expected %s header %q, got %q", header, want, got)
		}
	}

	if hsts := rr.Header().Get("Strict-Transport-Security"); hsts == "" {
		t.Error("expected Strict-Transport-Security header to be set, got empty")
	}
	if csp := rr.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("expected Content-Security-Policy header to be set, got empty")
	}
}

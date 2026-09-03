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

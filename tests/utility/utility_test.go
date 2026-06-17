package utility_test

import (
	"net/http"
	"net/http/httptest"
	"rtForum/utility"
	"testing"
)

func TestHashPasswordAndCheckPasswordHash(t *testing.T) {
	password := "my-secure-password"
	hash := utility.HashPassword(password)

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == password {
		t.Fatal("hash should not equal plaintext password")
	}
	if !utility.CheckPasswordHash(password, hash) {
		t.Fatal("expected password to match hash")
	}
}

func TestCheckPasswordHash_WrongPassword(t *testing.T) {
	hash := utility.HashPassword("correct-password")
	if utility.CheckPasswordHash("wrong-password", hash) {
		t.Fatal("expected wrong password to fail verification")
	}
}

func TestCheckCookieExist(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if utility.CheckCookieExist(httptest.NewRecorder(), req) {
		t.Fatal("expected no cookie initially")
	}

	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc"})
	if !utility.CheckCookieExist(httptest.NewRecorder(), req) {
		t.Fatal("expected session cookie to be detected")
	}
}

func TestCreateCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	utility.CreateCookie(rr, req)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "session_id" {
		t.Fatalf("expected session_id cookie, got %q", cookie.Name)
	}
	if cookie.Value == "" {
		t.Fatal("expected non-empty session id")
	}
	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}
	if !cookie.Secure {
		t.Fatal("expected Secure cookie")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("expected SameSiteStrictMode")
	}
}

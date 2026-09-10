package websocket_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"sync"
	"testing"
)

func TestCreateCategoryHandler_AsAdmin_Succeeds(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	body := `{"name":"Music"}`
	req := httptest.NewRequest(http.MethodPost, "/createCategory", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.CreateCategoryHandler)(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var created websocket.Category
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if created.Name != "Music" || created.ID == 0 {
		t.Fatalf("unexpected created category: %+v", created)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM category WHERE category_name = 'Music'`).Scan(&count); err != nil {
		t.Fatalf("failed to query category: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 category row, got %d", count)
	}
}

func TestCreateCategoryHandler_AsNonAdmin_Rejected(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-user", "alice", 2)

	body := `{"name":"Music"}`
	req := httptest.NewRequest(http.MethodPost, "/createCategory", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-user"})
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.CreateCategoryHandler)(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}
}

func TestCreateCategoryHandler_Unauthenticated_Rejected(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)

	body := `{"name":"Music"}`
	req := httptest.NewRequest(http.MethodPost, "/createCategory", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.CreateCategoryHandler)(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}
}

func TestCreateCategoryHandler_RejectsDuplicateName(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	body := `{"name":"Cuisine"}` // already seeded, see testutil.SetupForumDB
	req := httptest.NewRequest(http.MethodPost, "/createCategory", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.CreateCategoryHandler)(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rr.Code, rr.Body.String())
	}
}

// TestCreateCategoryHandler_ConcurrentCreatesSameName_OnlyOneSucceeds covers
// the actual bug: CreateCategoryHandler's COUNT(*)-then-INSERT is a TOCTOU
// race, so two concurrent requests for the same brand-new name could both
// pass the upfront check before either INSERT commits. What actually
// prevents the duplicate is idx_category_name_unique (see the migrate()
// backstop in database/sqlFuncs.go) plus this handler translating the
// resulting constraint error into the same 409 the sequential check gives —
// exactly one of these concurrent requests must succeed, the rest must see
// a clean 409, never a 500.
func TestCreateCategoryHandler_ConcurrentCreatesSameName_OnlyOneSucceeds(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	// database/sql's default unlimited pool would hand concurrent goroutines
	// separate connections — and a bare ":memory:" DSN (no shared cache)
	// gives each connection its own isolated database, silently defeating
	// this test (every goroutine would race against its own empty schema
	// instead of one shared one). Forcing a single shared connection is what
	// actually lets the goroutines below interleave against one real
	// database, the way concurrent requests through the production
	// connection pool do.
	db.SetMaxOpenConns(1)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	const n = 10
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := `{"name":"Astronomy"}`
			req := httptest.NewRequest(http.MethodPost, "/createCategory", bytes.NewBufferString(body))
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
			rr := httptest.NewRecorder()
			websocket.RequireAdmin(websocket.CreateCategoryHandler)(rr, req)
			codes[i] = rr.Code
		}(i)
	}
	wg.Wait()

	var created, conflicted int
	for i, code := range codes {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicted++
		default:
			t.Fatalf("request %d: unexpected status %d (want 201 or 409)", i, code)
		}
	}
	if created != 1 {
		t.Fatalf("expected exactly 1 request to succeed, got %d (of %d)", created, n)
	}
	if conflicted != n-1 {
		t.Fatalf("expected the other %d requests to see 409, got %d", n-1, conflicted)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM category WHERE category_name = 'Astronomy'`).Scan(&count); err != nil {
		t.Fatalf("failed to query category: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 stored category row, got %d", count)
	}
}

func TestCreateCategoryHandler_RejectsEmptyName(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	body := `{"name":"   "}`
	req := httptest.NewRequest(http.MethodPost, "/createCategory", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.CreateCategoryHandler)(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestEditCategoryHandler_AsAdmin_Renames(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	body := `{"id":1,"name":"Food"}` // id 1 is "Cuisine" in the seed data
	req := httptest.NewRequest(http.MethodPost, "/editCategory", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.EditCategoryHandler)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var name string
	if err := db.QueryRow(`SELECT category_name FROM category WHERE id = 1`).Scan(&name); err != nil {
		t.Fatalf("failed to query category: %v", err)
	}
	if name != "Food" {
		t.Fatalf("expected category 1 to be renamed to 'Food', got %q", name)
	}
}

func TestEditCategoryHandler_NotFound(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	body := `{"id":9999,"name":"Anything"}`
	req := httptest.NewRequest(http.MethodPost, "/editCategory", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.EditCategoryHandler)(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

func TestDeleteCategoryHandler_AsAdmin_RemovesCategoryAndRelations(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-admin", "admin", 1)

	// Category 1 ("Cuisine") has relations to posts 2 and 3 in the seed data.
	var relationsBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM category_relation WHERE category_id = 1`).Scan(&relationsBefore); err != nil {
		t.Fatalf("failed to query relations: %v", err)
	}
	if relationsBefore == 0 {
		t.Fatal("test assumption broken: expected seed data to have relations for category 1")
	}

	body := `{"id":1}`
	req := httptest.NewRequest(http.MethodPost, "/deleteCategory", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-admin"})
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.DeleteCategoryHandler)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var categoryCount, relationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM category WHERE id = 1`).Scan(&categoryCount); err != nil {
		t.Fatalf("failed to query category: %v", err)
	}
	if categoryCount != 0 {
		t.Fatal("expected the category to be deleted")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM category_relation WHERE category_id = 1`).Scan(&relationCount); err != nil {
		t.Fatalf("failed to query relations: %v", err)
	}
	if relationCount != 0 {
		t.Fatal("expected the category's relations to be deleted along with it")
	}
}

func TestDeleteCategoryHandler_AsNonAdmin_Rejected(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	websocket.AddAuthenticatedClient("session-user", "alice", 2)

	body := `{"id":1}`
	req := httptest.NewRequest(http.MethodPost, "/deleteCategory", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-user"})
	rr := httptest.NewRecorder()

	websocket.RequireAdmin(websocket.DeleteCategoryHandler)(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}
}

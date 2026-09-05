package websocket_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"rtForum/tests/testutil"
	"rtForum/websocket"
	"strconv"
	"strings"
	"testing"
)

// pngSignature is enough for http.DetectContentType to sniff "image/png" —
// the handler only sniffs the leading bytes, it doesn't validate a full
// image structure.
var pngSignature = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}

func newImageUploadRequest(t *testing.T, sessionID string, postID int, filename string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("post_id", strconv.Itoa(postID)); err != nil {
		t.Fatalf("failed to write post_id field: %v", err)
	}
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write file content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploadPostImage", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	}
	return req
}

// cleanUploads removes any files this test's handler calls wrote to disk —
// the handler resolves postImageUploadDir relative to the test binary's
// working directory, so leftover files would otherwise accumulate under
// tests/websocket/uploads across runs.
func cleanUploads(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { os.RemoveAll("./uploads") })
}

func TestUploadPostImageHandler_Success(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	cleanUploads(t)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	// Seeded post 1 is authored by actual_user (id 42).
	req := newImageUploadRequest(t, "session-actual", 1, "photo.png", pngSignature)
	rr := httptest.NewRecorder()
	websocket.UploadPostImageHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp struct {
		ImgURL string `json:"img_url"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.HasPrefix(resp.ImgURL, "/uploads/posts/") || !strings.HasSuffix(resp.ImgURL, ".png") {
		t.Fatalf("expected img_url under /uploads/posts/ with a .png extension, got %q", resp.ImgURL)
	}

	diskPath := filepath.Join("./uploads/posts", strings.TrimPrefix(resp.ImgURL, "/uploads/posts/"))
	if _, err := os.Stat(diskPath); err != nil {
		t.Fatalf("expected uploaded file to exist on disk at %s: %v", diskPath, err)
	}

	var storedImgURL string
	if err := db.QueryRow("SELECT img_url FROM post WHERE id = 1").Scan(&storedImgURL); err != nil {
		t.Fatalf("failed to query stored img_url: %v", err)
	}
	if storedImgURL != resp.ImgURL {
		t.Fatalf("expected post.img_url to be %q, got %q", resp.ImgURL, storedImgURL)
	}
}

func TestUploadPostImageHandler_ReplacingDeletesOldFile(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	cleanUploads(t)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	first := httptest.NewRecorder()
	websocket.UploadPostImageHandler(first, newImageUploadRequest(t, "session-actual", 1, "first.png", pngSignature))
	if first.Code != http.StatusOK {
		t.Fatalf("expected first upload to succeed, got %d: %s", first.Code, first.Body.String())
	}
	var firstResp struct {
		ImgURL string `json:"img_url"`
	}
	if err := json.NewDecoder(first.Body).Decode(&firstResp); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}
	firstPath := filepath.Join("./uploads/posts", strings.TrimPrefix(firstResp.ImgURL, "/uploads/posts/"))

	second := httptest.NewRecorder()
	websocket.UploadPostImageHandler(second, newImageUploadRequest(t, "session-actual", 1, "second.png", pngSignature))
	if second.Code != http.StatusOK {
		t.Fatalf("expected second upload to succeed, got %d: %s", second.Code, second.Body.String())
	}

	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Fatalf("expected the first uploaded file to be deleted after replacement, stat err: %v", err)
	}
}

func TestUploadPostImageHandler_RejectsNonImageContent(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	cleanUploads(t)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	req := newImageUploadRequest(t, "session-actual", 1, "notes.txt", []byte("just some plain text content"))
	rr := httptest.NewRecorder()
	websocket.UploadPostImageHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestUploadPostImageHandler_RejectsNonOwner(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	cleanUploads(t)
	websocket.AddAuthenticatedClient("session-alice", "alice", 2)

	// Seeded post 1 is authored by actual_user (id 42), not alice.
	req := newImageUploadRequest(t, "session-alice", 1, "photo.png", pngSignature)
	rr := httptest.NewRecorder()
	websocket.UploadPostImageHandler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, rr.Code, rr.Body.String())
	}
}

func TestUploadPostImageHandler_RejectsUnauthenticated(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	cleanUploads(t)

	req := newImageUploadRequest(t, "", 1, "photo.png", pngSignature)
	rr := httptest.NewRecorder()
	websocket.UploadPostImageHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}
}

func TestUploadPostImageHandler_RejectsNonexistentPost(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	cleanUploads(t)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	req := newImageUploadRequest(t, "session-actual", 99999, "photo.png", pngSignature)
	rr := httptest.NewRecorder()
	websocket.UploadPostImageHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

func TestUploadPostImageHandler_RejectsMissingFile(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	cleanUploads(t)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("post_id", "1"); err != nil {
		t.Fatalf("failed to write post_id field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/uploadPostImage", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-actual"})

	rr := httptest.NewRecorder()
	websocket.UploadPostImageHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestUploadPostImageHandler_CleansUpMultipartTempFiles exercises the exact
// condition that triggers Go's multipart reader to spill a form part to an
// on-disk temp file: total form data exceeding ParseMultipartForm's
// maxMemory argument (maxPostImageBytes, 5MiB). Without the handler calling
// r.MultipartForm.RemoveAll() afterward, such a temp file is never cleaned
// up.
func TestUploadPostImageHandler_CleansUpMultipartTempFiles(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	cleanUploads(t)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	before, err := filepath.Glob(filepath.Join(os.TempDir(), "multipart-*"))
	if err != nil {
		t.Fatalf("failed to glob temp dir before upload: %v", err)
	}

	content := append([]byte{}, pngSignature...)
	content = append(content, make([]byte, 5*1024*1024+512*1024)...)

	req := newImageUploadRequest(t, "session-actual", 1, "big.png", content)
	rr := httptest.NewRecorder()
	websocket.UploadPostImageHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	after, err := filepath.Glob(filepath.Join(os.TempDir(), "multipart-*"))
	if err != nil {
		t.Fatalf("failed to glob temp dir after upload: %v", err)
	}
	if len(after) > len(before) {
		t.Fatalf("expected no leftover multipart temp files after upload, before=%d after=%d (%v)", len(before), len(after), after)
	}
}

func TestAllPostsHandler_IncludesImgURL(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	cleanUploads(t)
	websocket.AddAuthenticatedClient("session-actual", "actual_user", 42)

	uploadRR := httptest.NewRecorder()
	websocket.UploadPostImageHandler(uploadRR, newImageUploadRequest(t, "session-actual", 1, "photo.png", pngSignature))
	if uploadRR.Code != http.StatusOK {
		t.Fatalf("expected upload to succeed, got %d: %s", uploadRR.Code, uploadRR.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/getAllPosts", nil)
	rr := httptest.NewRecorder()
	websocket.AllPostsHandler(rr, req)

	var posts []websocket.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Fatalf("failed to decode posts: %v", err)
	}
	var post1 *websocket.Post
	for i := range posts {
		if posts[i].PostId == 1 {
			post1 = &posts[i]
		}
	}
	if post1 == nil {
		t.Fatal("expected seeded post 1 in the response")
	}
	if post1.ImgURL == "" {
		t.Fatal("expected post 1 to have a non-empty ImgURL after upload")
	}

	// A post with no image should report an empty ImgURL, not null/missing.
	var post2 *websocket.Post
	for i := range posts {
		if posts[i].PostId == 2 {
			post2 = &posts[i]
		}
	}
	if post2 == nil {
		t.Fatal("expected seeded post 2 in the response")
	}
	if post2.ImgURL != "" {
		t.Fatalf("expected post 2 (no image uploaded) to have an empty ImgURL, got %q", post2.ImgURL)
	}
}

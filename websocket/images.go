package websocket

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"rtForum/database"
	"rtForum/utility"
	"strconv"
	"strings"
)

const (
	// postImageUploadDir is where post images are stored on disk — bind-
	// mounted in docker-compose.yml the same way ./database is, so images
	// survive a container recreate. postImageURLPrefix is the matching path
	// main.go serves them back under.
	postImageUploadDir = "./uploads/posts"
	postImageURLPrefix = "/uploads/posts/"

	maxPostImageBytes = 5 << 20 // 5 MiB
)

// allowedImageExtensions maps a sniffed (not client-claimed) content type to
// the extension its stored file gets — also doubles as the whitelist of
// accepted image types.
var allowedImageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadPostImageHandler attaches an image to a post the requester owns,
// replacing (and deleting from disk) any image the post already had.
// Expects multipart/form-data with a "post_id" field and an "image" file.
func UploadPostImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	client, err := authenticatedClientFromRequest(r)
	if err != nil {
		utility.ClearCookie(w)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Some slack over maxPostImageBytes for the multipart boundary/other
	// form fields — the image itself is still capped at maxPostImageBytes
	// by ParseMultipartForm's own accounting.
	r.Body = http.MaxBytesReader(w, r.Body, maxPostImageBytes+1<<20)
	if err := r.ParseMultipartForm(maxPostImageBytes); err != nil {
		http.Error(w, "image too large or malformed upload", http.StatusBadRequest)
		return
	}
	// ParseMultipartForm spills any part exceeding its maxMemory argument
	// (maxPostImageBytes here) to an on-disk temp file, which the caller is
	// responsible for removing — Go's own docs for MultipartForm call this
	// out. Uploads at or near the size limit routinely trigger the spill
	// (the field/boundary overhead alone can push a same-sized image past
	// the in-memory threshold), so without this every such upload leaked a
	// temp file with nothing to ever clean it up.
	defer r.MultipartForm.RemoveAll()

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil || postID <= 0 {
		http.Error(w, "a valid post_id is required", http.StatusBadRequest)
		return
	}

	var ownerID int
	var existingImgURL string
	err = database.ForumDB.QueryRow(
		"SELECT user_id, COALESCE(img_url, '') FROM post WHERE id = ?", postID,
	).Scan(&ownerID, &existingImgURL)
	if err == sql.ErrNoRows {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	} else if err != nil {
		slog.Error("failed to look up post owner for image upload", "error", err, "post_id", postID)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}
	if ownerID != client.userID {
		http.Error(w, "You can only add an image to your own posts", http.StatusForbidden)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "an image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sniff the real content type from the file's bytes rather than
	// trusting the client-supplied filename/MIME header, which is
	// attacker-controlled.
	sniff := make([]byte, 512)
	n, err := file.Read(sniff)
	if err != nil && err != io.EOF {
		slog.Error("failed to read uploaded image", "error", err)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}
	contentType := http.DetectContentType(sniff[:n])
	ext, ok := allowedImageExtensions[contentType]
	if !ok {
		http.Error(w, "unsupported image type: "+contentType, http.StatusBadRequest)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		slog.Error("failed to seek uploaded image", "error", err)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(postImageUploadDir, 0755); err != nil {
		slog.Error("failed to create upload directory", "error", err)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}

	filename, err := randomImageFilename(ext)
	if err != nil {
		slog.Error("failed to generate image filename", "error", err)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}
	destPath := filepath.Join(postImageUploadDir, filename)

	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		slog.Error("failed to create image file", "error", err, "path", destPath)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dest, file); err != nil {
		dest.Close()
		os.Remove(destPath)
		slog.Error("failed to write image file", "error", err, "path", destPath)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}
	if err := dest.Close(); err != nil {
		slog.Error("failed to close image file", "error", err, "path", destPath)
	}

	newImgURL := postImageURLPrefix + filename
	if _, err := database.ForumDB.Exec("UPDATE post SET img_url = ? WHERE id = ?", newImgURL, postID); err != nil {
		os.Remove(destPath)
		slog.Error("failed to update post img_url", "error", err, "post_id", postID)
		http.Error(w, "Failed to upload image", http.StatusInternalServerError)
		return
	}

	if existingImgURL != "" && existingImgURL != newImgURL {
		deleteUploadedImage(existingImgURL)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ImgURL string `json:"img_url"`
	}{ImgURL: newImgURL})
}

// randomImageFilename generates an unguessable filename so uploaded images
// can't collide with each other or be enumerated via the served /uploads/
// path.
func randomImageFilename(ext string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw) + ext, nil
}

// deleteUploadedImage removes a previously uploaded post image from disk,
// given the URL stored in post.img_url. Best-effort: failures are logged,
// not surfaced, since the DB row is already the source of truth and a
// leftover file on disk is a cleanup nicety, not a correctness issue.
func deleteUploadedImage(imgURL string) {
	if !strings.HasPrefix(imgURL, postImageURLPrefix) {
		return
	}
	filename := strings.TrimPrefix(imgURL, postImageURLPrefix)
	// Guard against a stored img_url that isn't a bare filename (it always
	// should be, since this handler is the only writer of the column) —
	// never let path.Join walk outside postImageUploadDir.
	if filename == "" || strings.ContainsAny(filename, "/\\") {
		return
	}
	path := filepath.Join(postImageUploadDir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Error("failed to remove old post image", "path", path, "error", err)
	}
}

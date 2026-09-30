package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"rtForum/database"
	"rtForum/utility"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// GetProfileHandler returns the requesting user's own editable profile
// fields — the settings page's initial load. Unlike ListUsersHandler this
// needs no RequireAdmin: any authenticated user may read their own profile,
// just never anyone else's (client.userID, from the verified session, is
// the only id ever queried here — never a client-supplied one).
func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	client, err := authenticatedClientFromRequest(r)
	if err != nil {
		utility.ClearCookie(w)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var fname, lname, email string
	if err := database.ForumDB.QueryRow("SELECT fname, lname, email FROM user WHERE id = ?", client.userID).Scan(&fname, &lname, &email); err != nil {
		slog.Error("failed to load profile", "error", err, "user_id", client.userID)
		http.Error(w, "Failed to load profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Fname string `json:"fname"`
		Lname string `json:"lname"`
		Email string `json:"email"`
	}{fname, lname, email})
}

// UpdateProfileHandler lets the requesting user change their own first
// name, last name, and email. Deliberately does not allow changing the
// username: post.author and category/chat lookups all key off it as a
// loose, string-based reference rather than a real FK to user.id (see
// createTables.sql), so a rename would need to cascade through data this
// handler has no reason to touch — out of scope for self-service editing.
func UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
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

	var requestBody struct {
		Fname string `json:"fname"`
		Lname string `json:"lname"`
		Email string `json:"email"`
	}
	if err := decodeJSONBody(w, r, &requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fname, lname, email, err := validateProfileUpdate(requestBody.Fname, requestBody.Lname, requestBody.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Same check-then-update-with-constraint-backstop pattern as
	// EditCategoryHandler: the COUNT(*) check gives a friendly error in the
	// common case, and user.email's UNIQUE constraint is the actual
	// TOCTOU-safe backstop against two concurrent requests racing onto the
	// same email. LOWER() on both sides — validateProfileUpdate already
	// lowercases `email`, but an older, pre-normalization row this is being
	// compared against might still be stored mixed-case (see migrate()'s
	// one-time backfill and validateRegistration's doc comment).
	var existing int
	if err := database.ForumDB.QueryRow("SELECT COUNT(*) FROM user WHERE LOWER(email) = ? AND id != ?", email, client.userID).Scan(&existing); err != nil {
		slog.Error("failed to check existing email", "error", err, "user_id", client.userID)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}
	if existing > 0 {
		http.Error(w, "That email is already in use", http.StatusConflict)
		return
	}

	if _, err := database.ForumDB.Exec("UPDATE user SET fname = ?, lname = ?, email = ? WHERE id = ?", fname, lname, email, client.userID); err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.Code == sqlite3.ErrConstraint {
			http.Error(w, "That email is already in use", http.StatusConflict)
			return
		}
		slog.Error("failed to update profile", "error", err, "user_id", client.userID)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	// Keep this session's cached identity in sync — (*Manager).checkLogin
	// serves client.email straight from memory, so without this the new
	// email wouldn't show up until the user's next full login even though
	// the database is already updated. Guarded by manager's lock, same as
	// checkLogin's own read of this field, since Client fields set after
	// creation aren't otherwise synchronized.
	manager.Lock()
	client.email = email
	manager.Unlock()

	slog.Info("profile updated", "user_id", client.userID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Profile updated"))
}

// UpdatePasswordHandler changes the requesting user's password, requiring
// their current one first — the only thing standing between "a stolen,
// still-valid session cookie" and "an attacker locks the real owner out by
// changing their password" is this check.
func UpdatePasswordHandler(w http.ResponseWriter, r *http.Request) {
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

	var requestBody struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSONBody(w, r, &requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validatePasswordChange(requestBody.CurrentPassword, requestBody.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var currentHash string
	if err := database.ForumDB.QueryRow("SELECT pass FROM user WHERE id = ?", client.userID).Scan(&currentHash); err != nil {
		slog.Error("failed to load password hash for update", "error", err, "user_id", client.userID)
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}
	if !utility.CheckPasswordHash(requestBody.CurrentPassword, currentHash) {
		http.Error(w, "Current password is incorrect", http.StatusBadRequest)
		return
	}

	newHash := utility.HashPassword(requestBody.NewPassword)
	if _, err := database.ForumDB.Exec("UPDATE user SET pass = ? WHERE id = ?", newHash, client.userID); err != nil {
		slog.Error("failed to update password", "error", err, "user_id", client.userID)
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	slog.Info("password changed", "user_id", client.userID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Password updated"))
}

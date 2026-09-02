package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"rtForum/database"
	"rtForum/utility"
)

// SearchMessagesHandler searches the content of every message in every
// conversation the requester belongs to — mirrors SearchPostsHandler's
// pattern (same query-length validation, same LIKE-escaping), but scoped to
// the authenticated requester's own conversations, since messages (unlike
// posts) are private.
func SearchMessagesHandler(w http.ResponseWriter, r *http.Request) {
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

	query, err := validateSearchQuery(r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	likePattern := "%" + escapeLikePattern(query) + "%"

	rows, err := database.ForumDB.Query(`
		SELECT message.id, message.conversation_id, user.uname, message.txt, message.created_at
		FROM message
		JOIN conversation_member ON conversation_member.conversation_id = message.conversation_id
			AND conversation_member.user_id = ?
		JOIN user ON user.id = message.sender_id
		WHERE message.txt LIKE ? ESCAPE '\'
		ORDER BY message.created_at DESC, message.id DESC
		LIMIT ?`, client.userID, likePattern, maxSearchResults)
	if err != nil {
		log.Printf("Error executing message search query: %s", err)
		http.Error(w, "Failed to search messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := []ChatHistoryMessage{}
	for rows.Next() {
		var m ChatHistoryMessage
		if err := rows.Scan(&m.Id, &m.ConversationID, &m.From, &m.Message, &m.CreatedAt); err != nil {
			log.Printf("Error scanning message search row: %s", err)
			http.Error(w, "Failed to search messages", http.StatusInternalServerError)
			return
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating message search rows: %s", err)
		http.Error(w, "Failed to search messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

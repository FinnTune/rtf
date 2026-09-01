package websocket

import (
	"encoding/json"
	"time"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error

const (
	// Event types
	GetChatHistory      = "get-chat-history"
	GetMoreChatHistory  = "get-more-chat-history"
	SendChatHistory     = "chat_history"
	EventReceiveMessage = "new-message"
	EventSendMessage    = "sent-message"
	UserConnect         = "user-connect"
	UsersList           = "users-online"
	Typing              = "typing"
	StopTyping          = "stop-typing"
	OpenDirectChat      = "open-direct-chat"
	CreateGroupChat     = "create-group-chat"
	ChatOpened          = "chat-opened"
	GetConversations    = "get-conversations"
	ConversationsList   = "conversations-list"
	ChatError           = "chat-error"
)

// ReceiveMessageEvent is the client->server "new-message" payload. Every
// message belongs to an existing conversation — a direct conversation with
// a given user has to be resolved/created first via "open-direct-chat" (and
// a group via "create-group-chat"), which is what hands the client the
// conversation_id it sends from here on.
type ReceiveMessageEvent struct {
	ConversationID int    `json:"conversation_id"`
	Message        string `json:"message"`
}

// SendMessageEvent is the server->client "sent-message" broadcast, sent
// the moment a message is stored.
type SendMessageEvent struct {
	ConversationID int       `json:"conversation_id"`
	From           string    `json:"from"`
	Message        string    `json:"message"`
	Sent           time.Time `json:"sent"`
}

// ChatHistoryMessage is one message in a "chat_history" response.
type ChatHistoryMessage struct {
	Id             int    `json:"id"`
	ConversationID int    `json:"conversation_id"`
	From           string `json:"from"`
	Message        string `json:"message"`
	CreatedAt      string `json:"created_at"`
}

// GetHistoryRequest is the "get-chat-history"/"get-more-chat-history" payload.
type GetHistoryRequest struct {
	ConversationID int `json:"conversation_id"`
	Limit          int `json:"limit"`
	Offset         int `json:"offset"`
}

// TypingEvent is the "typing"/"stop-typing" payload in both directions —
// From is set by the server on broadcast and ignored (never trusted) when
// received from a client.
type TypingEvent struct {
	ConversationID int    `json:"conversation_id"`
	From           string `json:"from,omitempty"`
}

// OpenDirectChatRequest is the "open-direct-chat" client->server payload.
type OpenDirectChatRequest struct {
	Username string `json:"username"`
}

// CreateGroupChatRequest is the "create-group-chat" client->server payload.
// The creator is added as a member automatically — they don't need to (and
// shouldn't have to) list themselves.
type CreateGroupChatRequest struct {
	Name      string   `json:"name"`
	Usernames []string `json:"usernames"`
}

// ConversationMember describes one participant of a conversation.
type ConversationMember struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
}

// ConversationInfo is one conversation's metadata — the "chat-opened"
// payload (a single conversation) and each element of "conversations-list".
type ConversationInfo struct {
	ConversationID int                  `json:"conversation_id"`
	IsGroup        bool                 `json:"is_group"`
	Name           string               `json:"name,omitempty"`
	Members        []ConversationMember `json:"members"`
}

// ChatErrorEvent is sent back to a single requesting client (never
// broadcast) when a chat action can't be completed — e.g. a group chat
// created with an unresolvable username.
type ChatErrorEvent struct {
	Message string `json:"message"`
}

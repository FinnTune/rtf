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
	MarkRead            = "mark-read"
	ReadReceipt         = "read-receipt"
	MessageAck          = "message-ack"
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
// the moment a message is stored. Id is the message's real database id —
// needed by the recipient's client to mark it read via "mark-read".
type SendMessageEvent struct {
	Id             int       `json:"id"`
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
	// Every member's current read watermark, so a freshly opened window
	// knows "seen by" state immediately without waiting for a live
	// "read-receipt" event.
	ReadStates []ReadState `json:"read_states"`
}

// ReadState is one conversation member's "read up to" watermark.
// LastReadMessageID is 0 for a member who hasn't read anything yet — there
// being no message_read row at all is indistinguishable from (and treated
// the same as) an explicit watermark of 0.
type ReadState struct {
	UserID            int    `json:"user_id"`
	Username          string `json:"username"`
	LastReadMessageID int    `json:"last_read_message_id"`
}

// ChatErrorEvent is sent back to a single requesting client (never
// broadcast) when a chat action can't be completed — e.g. a group chat
// created with an unresolvable username.
type ChatErrorEvent struct {
	Message string `json:"message"`
}

// MarkReadRequest is the "mark-read" client->server payload — the client
// reports having seen everything up to and including MessageID.
type MarkReadRequest struct {
	ConversationID int `json:"conversation_id"`
	MessageID      int `json:"message_id"`
}

// ReadReceiptEvent is the "read-receipt" server->other-members broadcast,
// sent whenever a member's read watermark advances.
type ReadReceiptEvent struct {
	ConversationID int    `json:"conversation_id"`
	UserID         int    `json:"user_id"`
	Username       string `json:"username"`
	MessageID      int    `json:"message_id"`
}

// MessageAckEvent is sent back to just the sender's own connection right
// after their message is stored — "sent-message" is only ever broadcast to
// a conversation's OTHER members (see sendMessage in ws-manager.go), so
// without this the sender would never learn their own message's real,
// database-assigned id and could never see a "seen by" indicator advance
// past their own latest message.
type MessageAckEvent struct {
	ConversationID int `json:"conversation_id"`
	Id             int `json:"id"`
}

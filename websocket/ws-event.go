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
	PostReactionUpdated = "post-reaction-updated"
	CommentDeleted      = "comment-deleted"
	CommentEdited       = "comment-edited"
	PostEdited          = "post-edited"
	CommentAdded        = "comment-added"
)

// ReceiveMessageEvent is the client->server "new-message" payload. Every
// message belongs to an existing conversation — a direct conversation with
// a given user has to be resolved/created first via "open-direct-chat" (and
// a group via "create-group-chat"), which is what hands the client the
// conversation_id it sends from here on.
//
// ClientMsgID is an opaque token the client generates and this server never
// interprets — it's echoed back verbatim in the resulting message-ack (on
// success) or chat-error (on a validation failure) so the client can
// reconcile its own optimistic local echo of this specific message, rather
// than guessing "the oldest still-unconfirmed one", which breaks the moment
// more than one send is outstanding at once (e.g. an earlier send silently
// dropped - see sendMessage's membership/rate-limit checks - leaves a
// permanently-unconfirmed local echo that the next real ack would
// otherwise misattach itself to).
type ReceiveMessageEvent struct {
	ConversationID int    `json:"conversation_id"`
	Message        string `json:"message"`
	ClientMsgID    string `json:"client_msg_id,omitempty"`
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
//
// ClientMsgID is only ever set when this error is rejecting a specific
// "new-message" send (echoing that request's ClientMsgID) — every other
// sendChatError call site (open-direct-chat, create-group-chat, ...) has no
// particular message to correlate to, so it's left empty there.
type ChatErrorEvent struct {
	Message     string `json:"message"`
	ClientMsgID string `json:"client_msg_id,omitempty"`
}

// MarkReadRequest is the "mark-read" client->server payload — the client
// reports having seen everything up to and including MessageID.
type MarkReadRequest struct {
	ConversationID int `json:"conversation_id"`
	MessageID      int `json:"message_id"`
}

// PostReactionUpdatedEvent is broadcast to every connected client whenever
// a post's aggregate reaction counts change, so a post visible in another
// client's feed or single-post view doesn't go stale until they reload.
// Deliberately carries no personal "my reaction" field — that's only ever
// known and updated for the client that actually performed the action, via
// ReactToPostHandler's own HTTP response to them.
type PostReactionUpdatedEvent struct {
	PostID       int `json:"post_id"`
	LikeCount    int `json:"like_count"`
	DislikeCount int `json:"dislike_count"`
}

// CommentDeletedEvent is broadcast to every connected client whenever a
// comment is deleted, so a comment list visible in another client's
// single-post view doesn't keep showing a comment that's actually gone
// until they happen to reload.
type CommentDeletedEvent struct {
	PostID    int `json:"post_id"`
	CommentID int `json:"comment_id"`
}

// CommentEditedEvent is broadcast to every connected client whenever a
// comment's content is edited, so a comment list visible in another
// client's single-post view doesn't keep showing the stale pre-edit text
// until they happen to reload.
type CommentEditedEvent struct {
	PostID    int    `json:"post_id"`
	CommentID int    `json:"comment_id"`
	Content   string `json:"content"`
}

// PostEditedEvent is broadcast to every connected client whenever a post's
// title/content is edited, so a permalink (SinglePostView) open in another
// client on the same post doesn't keep showing the stale pre-edit text
// until they happen to reload.
type PostEditedEvent struct {
	PostID  int    `json:"post_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
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
//
// ClientMsgID echoes the originating ReceiveMessageEvent's token, so the
// client can reconcile the specific local echo this ack confirms rather
// than assuming "the oldest unconfirmed one" (see ReceiveMessageEvent's doc
// comment).
type MessageAckEvent struct {
	ConversationID int    `json:"conversation_id"`
	Id             int    `json:"id"`
	ClientMsgID    string `json:"client_msg_id,omitempty"`
}

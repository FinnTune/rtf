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
	EventReceiveMessage = "new-message"
	EventSendMessage    = "sent-message"
	UserConnect         = "user-connect"
	UsersList           = "users-online"
	PMessage            = "private-message"
	SendChatHistory     = "chat_history"
	Typing              = "typing"
	StopTyping          = "stop-typing"
	GetMoreChatHistory  = "get-more-chat-history"
)

type ReceiveMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
	To      string `json:"to"`
}

type SendMessageEvent struct {
	ReceiveMessageEvent
	Sent time.Time `json:"sent"`
}

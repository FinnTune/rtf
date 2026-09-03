// Post's JSON keys have no server-side struct tags — these field names are
// the literal wire format (websocket/structs.go's Post struct). MyReaction
// is "none" | "liked" | "disliked" — "none" for both an anonymous viewer
// and a logged-in one who hasn't reacted (the two are indistinguishable
// from the response alone, since post-listing endpoints don't require
// login just to browse).
export interface Post {
  PostId: number
  UserId: number
  Title: string
  Content: string
  Author: string
  Created: string
  // Empty string means "no image" — never null/undefined, see attachReactionData's counterpart on the Go side.
  ImgURL: string
  LikeCount: number
  DislikeCount: number
  MyReaction: string
}

export interface Category {
  id: number
  name: string
}

export interface Comment {
  username: string
  id: number
  user_id: number
  post_id: number
  content: string
  created_at: string
}

export interface AuthUser {
  id: number
  username: string
  email: string
  joined: string
  otp: string
  // UX-only (shows/hides admin views) — the backend re-verifies this from
  // the database on every admin-gated request regardless of what the
  // client sends, so there's no security concern in trusting it here.
  role: string
}

// Matches websocket/structs.go's RegUser JSON tags exactly.
export interface RegisterPayload {
  fname: string
  lname: string
  uname: string
  email: string
  age: string
  gender: string
  password: string
}

// Normalized client-side shape for a chat message, regardless of which of
// the two different wire shapes it came from: a real-time `sent-message`
// event (`{message, from, sent}`, `sent` an RFC3339 string) or a
// `chat_history` entry (`{..., message, created_at}`, created_at a SQLite
// datetime string) — `timestamp` holds whichever of those two the message
// came with, always valid input to `new Date(...)`. Which conversation a
// message belongs to is which ChatWindowState it lives in, not a field on
// the message itself.
// id is 0 for a message this client sent optimistically and hasn't yet
// reconciled with the server's real id (the server never echoes a sent
// message back to its own sender) — never a real message's actual id,
// which is always a positive, database-assigned integer.
export interface ChatMessageVM {
  id: number
  from: string
  message: string
  timestamp: string | number
}

// Matches websocket/ws-event.go's ConversationMember/ConversationInfo JSON
// tags exactly.
export interface ConversationMember {
  user_id: number
  username: string
}

export interface ReadState {
  user_id: number
  username: string
  last_read_message_id: number
}

export interface ConversationInfo {
  conversation_id: number
  is_group: boolean
  name?: string
  members: ConversationMember[]
  read_states: ReadState[]
}

// Matches websocket.ChatHistoryMessage's JSON tags — the /searchMessages
// REST endpoint returns this same shape.
export interface MessageSearchResult {
  id: number
  conversation_id: number
  from: string
  message: string
  created_at: string
}

// /listUsers's per-user shape (admin-only) — never includes a password field.
export interface UserSummary {
  id: number
  username: string
  email: string
  role: string
  banned: boolean
}

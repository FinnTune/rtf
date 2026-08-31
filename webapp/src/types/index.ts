// Post's JSON keys have no server-side struct tags — these field names are
// the literal wire format (websocket/structs.go's Post struct).
export interface Post {
  PostId: number
  UserId: number
  Title: string
  Content: string
  Author: string
  Created: string
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
// event (`{message, from, to, sent}`, `sent` an RFC3339 string) or a
// `chat_history` entry (`{..., message, created_at}`, created_at a SQLite
// datetime string) — `timestamp` holds whichever of those two the message
// came with, always valid input to `new Date(...)`.
export interface ChatMessageVM {
  from: string
  to: string
  message: string
  timestamp: string | number
}

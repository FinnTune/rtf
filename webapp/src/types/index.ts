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

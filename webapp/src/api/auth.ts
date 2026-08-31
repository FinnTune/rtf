import { jsonHeaders, requestJson, requestVoid } from './client'
import type { AuthUser, RegisterPayload } from '../types'

// Raw /checkLogin and /login response shape (websocket/structs.go's
// UserLoginResponse) — every field but `loggedIn` is optional because a
// logged-out /checkLogin response only ever sets `loggedIn: false`.
interface LoginResponse {
  loggedIn: boolean
  id?: number
  username?: string
  email?: string
  joined?: string
  otp?: string
  role?: string
}

function toAuthUser(data: LoginResponse): AuthUser | null {
  if (data.id === undefined || !data.username || !data.email || !data.joined || !data.otp) {
    return null
  }
  return {
    id: data.id,
    username: data.username,
    email: data.email,
    joined: data.joined,
    otp: data.otp,
    role: data.role ?? 'user',
  }
}

export async function checkLogin(): Promise<AuthUser | null> {
  const { data } = await requestJson<LoginResponse>('/checkLogin', { method: 'POST', headers: jsonHeaders })
  if (!data.loggedIn) {
    return null
  }
  return toAuthUser(data)
}

export async function login(username: string, password: string): Promise<AuthUser> {
  const { data } = await requestJson<LoginResponse>('/login', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ username, password }),
  })
  // /login's `loggedIn` field is not meaningful — the backend always sends
  // `false` here even on success (the session is already marked logged-in
  // server-side by this point). Success is signaled by the HTTP status
  // alone, same as the vanilla-JS app relied on.
  const user = toAuthUser(data)
  if (!user) {
    throw new Error('Unexpected response from server.')
  }
  return user
}

export async function register(payload: RegisterPayload): Promise<void> {
  await requestVoid('/register', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify(payload),
  })
}

export async function logout(): Promise<void> {
  await requestVoid('/logout', { method: 'POST', headers: jsonHeaders })
}

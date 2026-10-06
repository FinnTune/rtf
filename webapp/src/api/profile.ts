import { jsonHeaders, requestJson, requestVoid } from './client'

export interface Profile {
  fname: string
  lname: string
  email: string
}

export async function getProfile(): Promise<Profile> {
  const { data } = await requestJson<Profile>('/profile')
  return data
}

export async function updateProfile(profile: Profile): Promise<void> {
  await requestVoid('/updateProfile', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify(profile),
  })
}

export async function updatePassword(currentPassword: string, newPassword: string): Promise<void> {
  await requestVoid('/updatePassword', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  })
}

// Matches websocket/handlers.go's GetUserProfileHandler JSON tags exactly.
// Distinct from Profile above: this is a PUBLIC summary of any username
// (join date, post/comment counts), not the authenticated caller's own
// editable fname/lname/email.
export interface UserProfile {
  username: string
  joined: string
  post_count: number
  comment_count: number
}

export async function getUserProfile(username: string): Promise<UserProfile> {
  const { data } = await requestJson<UserProfile>(`/getUserProfile?username=${encodeURIComponent(username)}`)
  return data
}

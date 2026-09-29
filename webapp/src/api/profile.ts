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

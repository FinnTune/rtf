import { jsonHeaders, requestJson, requestVoid } from './client'
import type { UserSummary } from '../types'

interface UserPage {
  users: UserSummary[]
  total: number | null
}

export async function listUsers(offset: number, limit: number): Promise<UserPage> {
  const { data, total } = await requestJson<UserSummary[]>(`/listUsers?limit=${limit}&offset=${offset}`)
  return { users: data, total }
}

export async function setUserBanned(userId: number, banned: boolean): Promise<void> {
  await requestVoid('/setUserBanned', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ user_id: userId, banned }),
  })
}

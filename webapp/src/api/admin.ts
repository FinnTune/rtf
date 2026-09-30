import { jsonHeaders, requestJson, requestVoid } from './client'
import type { UserSummary } from '../types'

interface UserPage {
  users: UserSummary[]
  total: number | null
}

export async function listUsers(offset: number, limit: number, query?: string): Promise<UserPage> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  if (query) params.set('q', query)
  const { data, total } = await requestJson<UserSummary[]>(`/listUsers?${params.toString()}`)
  return { users: data, total }
}

export async function setUserBanned(userId: number, banned: boolean): Promise<void> {
  await requestVoid('/setUserBanned', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ user_id: userId, banned }),
  })
}

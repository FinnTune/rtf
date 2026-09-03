import { jsonHeaders, requestJson, requestVoid } from './client'
import type { UserSummary } from '../types'

export async function listUsers(): Promise<UserSummary[]> {
  const { data } = await requestJson<UserSummary[]>('/listUsers')
  return data
}

export async function setUserBanned(userId: number, banned: boolean): Promise<void> {
  await requestVoid('/setUserBanned', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ user_id: userId, banned }),
  })
}

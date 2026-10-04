import { jsonHeaders, requestJson, requestVoid } from './client'
import type { NotificationItem } from '../types'

export async function getNotifications(offset: number, limit: number): Promise<{ notifications: NotificationItem[]; unreadCount: number; total: number }> {
  const { data, total } = await requestJson<{ notifications: NotificationItem[]; unread_count: number }>(
    `/notifications?limit=${limit}&offset=${offset}`,
  )
  return { notifications: data.notifications, unreadCount: data.unread_count, total: total ?? 0 }
}

// Marks one notification read (id given) or every unread one (omitted) —
// see MarkNotificationsReadHandler's own doc comment for why a wrong-owner
// id silently no-ops rather than erroring.
export async function markNotificationsRead(id?: number): Promise<void> {
  await requestVoid('/markNotificationsRead', {
    method: 'POST',
    headers: jsonHeaders,
    body: id === undefined ? undefined : JSON.stringify({ id }),
  })
}

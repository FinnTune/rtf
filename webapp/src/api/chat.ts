import { requestJson } from './client'
import type { MessageSearchResult } from '../types'

// Unlike post search, this is scoped server-side to the requester's own
// conversations (messages are private) — see SearchMessagesHandler.
export async function searchMessages(query: string): Promise<MessageSearchResult[]> {
  const { data } = await requestJson<MessageSearchResult[]>(`/searchMessages?q=${encodeURIComponent(query)}`)
  return data
}

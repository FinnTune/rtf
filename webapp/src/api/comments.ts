import { jsonHeaders, requestJson, requestVoid } from './client'
import type { Comment } from '../types'

export async function getComments(postId: number, offset: number, limit: number): Promise<{ comments: Comment[]; total: number }> {
  const { data, total } = await requestJson<Comment[]>(`/comments?postId=${postId}&limit=${limit}&offset=${offset}`)
  return { comments: data, total: total ?? 0 }
}

export async function addComment(postId: number, content: string): Promise<Comment> {
  const { data } = await requestJson<Comment>('/addcomment', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ post_id: postId, content }),
  })
  return data
}

export async function editComment(id: number, content: string): Promise<{ content: string }> {
  const { data } = await requestJson<{ content: string }>('/editComment', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ id, content }),
  })
  return data
}

export async function deleteComment(id: number): Promise<void> {
  await requestVoid('/deleteComment', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ id }),
  })
}

import { jsonHeaders, requestJson, requestVoid } from './client'
import type { Post } from '../types'

export interface PostCategoryRef {
  id: number
  name: string
}

interface PostPage {
  posts: Post[]
  total: number | null
}

// Mirrors the backend's validSortValues in websocket/validate.go.
export type PostSort = 'newest' | 'most_liked' | 'most_commented'

export async function getAllPosts(offset: number, limit: number, sort: PostSort = 'newest'): Promise<PostPage> {
  const { data, total } = await requestJson<Post[]>(`/getAllPosts?limit=${limit}&offset=${offset}&sort=${sort}`)
  return { posts: data, total }
}

export async function getPostsByAuthor(
  author: string,
  offset: number,
  limit: number,
  sort: PostSort = 'newest',
): Promise<PostPage> {
  const { data, total } = await requestJson<Post[]>(
    `/getPostsByAuthor?author=${encodeURIComponent(author)}&limit=${limit}&offset=${offset}&sort=${sort}`,
  )
  return { posts: data, total }
}

export async function getPostsByCategory(
  categories: string[],
  offset: number,
  limit: number,
  sort: PostSort = 'newest',
): Promise<PostPage> {
  const { data, total } = await requestJson<Post[]>(`/getPostsByCategory?limit=${limit}&offset=${offset}&sort=${sort}`, {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ categories }),
  })
  return { posts: data, total }
}

// The backend doesn't paginate search results (no limit/offset params, no
// X-Total-Count) — total comes back null, same as the original app, which
// never showed pagination controls for a search.
export async function searchPosts(query: string, sort: PostSort = 'newest'): Promise<PostPage> {
  const { data, total } = await requestJson<Post[]>(`/searchPosts?q=${encodeURIComponent(query)}&sort=${sort}`)
  return { posts: data, total }
}

export async function getPost(id: number | string): Promise<Post> {
  const { data } = await requestJson<Post>(`/getPost?id=${id}`)
  return data
}

// Returns the new post's id so the caller can immediately attach an image
// via uploadPostImage, a separate request since image upload is multipart
// and post creation is JSON.
export async function addPost(title: string, content: string, categories: PostCategoryRef[]): Promise<number> {
  const { data } = await requestJson<{ id: number }>('/addPost', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ title, content, categories }),
  })
  return data.id
}

export async function editPost(
  id: number,
  title: string,
  content: string,
  categories: PostCategoryRef[],
): Promise<{ title: string; content: string }> {
  const { data } = await requestJson<{ title: string; content: string }>('/editPost', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ id, title, content, categories }),
  })
  return data
}

export async function deletePost(id: number): Promise<void> {
  await requestVoid('/deletePost', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ id }),
  })
}

// Uploads (or replaces) the image attached to a post the caller owns. No
// Content-Type header is set — the browser fills in the multipart boundary
// itself from the FormData body.
export async function uploadPostImage(postId: number, file: File): Promise<string> {
  const formData = new FormData()
  formData.append('post_id', String(postId))
  formData.append('image', file)
  const { data } = await requestJson<{ img_url: string }>('/uploadPostImage', {
    method: 'POST',
    body: formData,
  })
  return data.img_url
}

export interface ReactionResult {
  likeCount: number
  dislikeCount: number
  myReaction: string
}

// Submitting the same reaction again toggles it off; submitting the
// opposite one switches it — see ReactToPostHandler for the exact
// server-side state machine this mirrors for the optimistic update in
// hooks/useReaction.ts.
export async function reactToPost(postId: number, isLiked: boolean): Promise<ReactionResult> {
  const { data } = await requestJson<{ like_count: number; dislike_count: number; my_reaction: string }>('/reactToPost', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ post_id: postId, is_liked: isLiked }),
  })
  return { likeCount: data.like_count, dislikeCount: data.dislike_count, myReaction: data.my_reaction }
}

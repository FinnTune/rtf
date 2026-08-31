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

export async function getAllPosts(offset: number, limit: number): Promise<PostPage> {
  const { data, total } = await requestJson<Post[]>(`/getAllPosts?limit=${limit}&offset=${offset}`)
  return { posts: data, total }
}

export async function getPostsByAuthor(author: string, offset: number, limit: number): Promise<PostPage> {
  const { data, total } = await requestJson<Post[]>(
    `/getPostsByAuthor?author=${encodeURIComponent(author)}&limit=${limit}&offset=${offset}`,
  )
  return { posts: data, total }
}

export async function getPostsByCategory(categories: string[], offset: number, limit: number): Promise<PostPage> {
  const { data, total } = await requestJson<Post[]>(`/getPostsByCategory?limit=${limit}&offset=${offset}`, {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ categories }),
  })
  return { posts: data, total }
}

// The backend doesn't paginate search results (no limit/offset params, no
// X-Total-Count) — total comes back null, same as the original app, which
// never showed pagination controls for a search.
export async function searchPosts(query: string): Promise<PostPage> {
  const { data, total } = await requestJson<Post[]>(`/searchPosts?q=${encodeURIComponent(query)}`)
  return { posts: data, total }
}

export async function getPost(id: number | string): Promise<Post> {
  const { data } = await requestJson<Post>(`/getPost?id=${id}`)
  return data
}

export async function addPost(title: string, content: string, categories: PostCategoryRef[]): Promise<void> {
  await requestVoid('/addPost', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ title, content, categories }),
  })
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

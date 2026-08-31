import { requestJson } from './client'
import type { Category } from '../types'

// Categories change rarely, and several components on the same page need
// them (the sidebar nav, the add/edit-post category picker) — cache the
// in-flight/resolved request rather than each caller re-fetching.
let categoriesPromise: Promise<Category[]> | null = null

export function getCategories(): Promise<Category[]> {
  if (!categoriesPromise) {
    categoriesPromise = requestJson<Category[]>('/getCategories').then(({ data }) => data)
    // Don't cache a failure — let the next caller retry instead of being
    // stuck with a permanently-rejected cached promise.
    categoriesPromise.catch(() => {
      categoriesPromise = null
    })
  }
  return categoriesPromise
}

export async function getPostCategories(postId: number): Promise<Category[]> {
  const { data } = await requestJson<Category[]>(`/getPostCategories?postId=${postId}`)
  return data
}

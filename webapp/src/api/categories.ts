import { jsonHeaders, requestJson, requestVoid } from './client'
import type { Category } from '../types'

// Categories change rarely, and several components on the same page need
// them (the sidebar nav, the add/edit-post category picker) — cache the
// in-flight/resolved request rather than each caller re-fetching.
let categoriesPromise: Promise<Category[]> | null = null

// Components that hold their own copy of the list in state (CategoryNav,
// CategoryPicker) need to know when a mutation below invalidates the cache,
// not just that the cache itself is stale — otherwise an already-mounted
// component (the sidebar nav in particular, which never remounts as you
// navigate around the app) keeps showing what it fetched on mount
// indefinitely, even though the next fresh getCategories() call elsewhere
// would return the up-to-date list.
type Listener = () => void
const listeners = new Set<Listener>()

export function subscribeToCategoryChanges(listener: Listener): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

function invalidateCategoriesCache() {
  categoriesPromise = null
  listeners.forEach((listener) => listener())
}

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

export async function createCategory(name: string): Promise<Category> {
  const { data } = await requestJson<Category>('/createCategory', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ name }),
  })
  invalidateCategoriesCache()
  return data
}

export async function editCategory(id: number, name: string): Promise<Category> {
  const { data } = await requestJson<Category>('/editCategory', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ id, name }),
  })
  invalidateCategoriesCache()
  return data
}

export async function deleteCategory(id: number): Promise<void> {
  await requestVoid('/deleteCategory', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ id }),
  })
  invalidateCategoriesCache()
}

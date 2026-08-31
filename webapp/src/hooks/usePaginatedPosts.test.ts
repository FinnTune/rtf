import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider } from '../contexts/StatusMessageContext'
import type { Post } from '../types'
import { usePaginatedPosts } from './usePaginatedPosts'

function makePost(id: number): Post {
  return { PostId: id, UserId: 1, Title: `Post ${id}`, Content: 'x', Author: 'alice', Created: '2026-01-01' }
}

describe('usePaginatedPosts', () => {
  it('loads page 1 on mount', async () => {
    const fetcher = vi.fn().mockResolvedValue({ posts: [makePost(1), makePost(2)], total: 5 })
    const { result } = renderHook(() => usePaginatedPosts(fetcher, [], 2), { wrapper: StatusMessageProvider })

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(fetcher).toHaveBeenCalledWith(0, 2)
    expect(result.current.posts).toHaveLength(2)
    expect(result.current.total).toBe(5)
    expect(result.current.offset).toBe(0)
  })

  it('goToOffset fetches the requested page', async () => {
    const fetcher = vi.fn().mockResolvedValue({ posts: [makePost(1)], total: 5 })
    const { result } = renderHook(() => usePaginatedPosts(fetcher, [], 2), { wrapper: StatusMessageProvider })
    await waitFor(() => expect(result.current.loading).toBe(false))

    fetcher.mockResolvedValue({ posts: [makePost(3), makePost(4)], total: 5 })
    act(() => result.current.goToOffset(2))
    await waitFor(() => expect(result.current.offset).toBe(2))
    expect(fetcher).toHaveBeenCalledWith(2, 2)
  })

  it('snaps back a page when the requested page comes back empty', async () => {
    const fetcher = vi.fn().mockResolvedValue({ posts: [makePost(1), makePost(2)], total: 4 })
    const { result } = renderHook(() => usePaginatedPosts(fetcher, [], 2), { wrapper: StatusMessageProvider })
    await waitFor(() => expect(result.current.loading).toBe(false))

    // Page at offset 2 (page 2) is now empty — e.g. its posts were deleted.
    fetcher.mockResolvedValueOnce({ posts: [], total: 2 }).mockResolvedValueOnce({ posts: [makePost(1)], total: 2 })
    act(() => result.current.goToOffset(2))

    await waitFor(() => expect(result.current.offset).toBe(0))
    expect(fetcher).toHaveBeenCalledWith(2, 2)
    expect(fetcher).toHaveBeenCalledWith(0, 2)
  })

  it('re-fetches from page 1 when deps change', async () => {
    const fetcher = vi.fn().mockResolvedValue({ posts: [makePost(1)], total: 1 })
    const { result, rerender } = renderHook(({ dep }: { dep: string }) => usePaginatedPosts(fetcher, [dep], 10), {
      wrapper: StatusMessageProvider,
      initialProps: { dep: 'Cuisine' },
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(fetcher).toHaveBeenCalledTimes(1)

    rerender({ dep: 'Sports' })
    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2))
    expect(fetcher).toHaveBeenLastCalledWith(0, 10)
  })

  it('reports total as null through unchanged (e.g. search has no total)', async () => {
    const fetcher = vi.fn().mockResolvedValue({ posts: [makePost(1)], total: null })
    const { result } = renderHook(() => usePaginatedPosts(fetcher, [], 10), { wrapper: StatusMessageProvider })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.total).toBeNull()
  })

  it('shows a status message and stops loading on a fetch error', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('Failed to load posts (500)'))
    const { result } = renderHook(() => usePaginatedPosts(fetcher, [], 10), { wrapper: StatusMessageProvider })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.posts).toEqual([])
  })
})

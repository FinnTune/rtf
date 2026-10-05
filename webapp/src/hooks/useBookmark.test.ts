import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider } from '../contexts/StatusMessageContext'
import type { Post } from '../types'
import { useBookmark } from './useBookmark'

vi.mock('../api/posts', () => ({ bookmarkPost: vi.fn() }))
import { bookmarkPost } from '../api/posts'

function makePost(overrides: Partial<Post> = {}): Post {
  return {
    PostId: 1,
    UserId: 1,
    Title: 'Test',
    Content: 'Body',
    Author: 'alice',
    Created: '2026-01-01',
    ImgURL: '',
    LikeCount: 0,
    DislikeCount: 0,
    MyReaction: 'none',
    MyBookmark: false,
    ...overrides,
  }
}

beforeEach(() => {
  vi.mocked(bookmarkPost).mockReset()
})

describe('useBookmark', () => {
  it('starts from the post prop\'s MyBookmark value', () => {
    const { result } = renderHook(() => useBookmark(makePost({ MyBookmark: true })), { wrapper: StatusMessageProvider })
    expect(result.current.bookmarked).toBe(true)
  })

  it('optimistically flips before the request resolves, then reconciles with the server response', async () => {
    let resolveFetch!: (value: boolean) => void
    vi.mocked(bookmarkPost).mockReturnValue(new Promise((resolve) => (resolveFetch = resolve)))
    const { result } = renderHook(() => useBookmark(makePost({ MyBookmark: false })), { wrapper: StatusMessageProvider })

    act(() => result.current.toggle())
    expect(result.current.bookmarked).toBe(true)
    expect(result.current.pending).toBe(true)

    await act(async () => resolveFetch(true))
    expect(bookmarkPost).toHaveBeenCalledWith(1)
    expect(result.current.bookmarked).toBe(true)
    expect(result.current.pending).toBe(false)
  })

  it('rolls back to the previous state and shows an error if the request fails', async () => {
    vi.mocked(bookmarkPost).mockRejectedValue(new Error('Failed to update bookmark'))
    const { result } = renderHook(() => useBookmark(makePost({ MyBookmark: false })), { wrapper: StatusMessageProvider })

    act(() => result.current.toggle())
    expect(result.current.bookmarked).toBe(true)

    await waitFor(() => expect(result.current.pending).toBe(false))
    expect(result.current.bookmarked).toBe(false)
  })

  it('reconciles to false if toggling off (the server is the source of truth, not a blind flip)', async () => {
    vi.mocked(bookmarkPost).mockResolvedValue(false)
    const { result } = renderHook(() => useBookmark(makePost({ MyBookmark: true })), { wrapper: StatusMessageProvider })

    act(() => result.current.toggle())
    expect(result.current.bookmarked).toBe(false)

    await waitFor(() => expect(result.current.pending).toBe(false))
    expect(bookmarkPost).toHaveBeenCalledWith(1)
    expect(result.current.bookmarked).toBe(false)
  })
})

import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider } from '../contexts/StatusMessageContext'
import type { Post } from '../types'
import { useReaction } from './useReaction'

vi.mock('../api/posts', () => ({ reactToPost: vi.fn() }))
import { reactToPost } from '../api/posts'

// Most of useReaction's optimistic/rollback/live-update behavior is already
// exercised indirectly (and thoroughly) through ReactionButtons.test.tsx.
// This file covers applyOptimistic's remaining untested branch: toggling an
// *existing* reaction off again (as opposed to setting one from none, or
// switching between the two), in both directions.

function makePost(overrides: Partial<Post> = {}): Post {
  return {
    PostId: 1,
    UserId: 1,
    Title: 'Test',
    Content: 'Body',
    Author: 'alice',
    Created: '2026-01-01',
    ImgURL: '',
    LikeCount: 3,
    DislikeCount: 2,
    MyReaction: 'none',
    ...overrides,
  }
}

beforeEach(() => {
  vi.mocked(reactToPost).mockReset()
})

describe('useReaction', () => {
  it('toggles the reaction off (and decrements, not increments) when clicking Like again while already liked', async () => {
    vi.mocked(reactToPost).mockResolvedValue({ likeCount: 2, dislikeCount: 2, myReaction: 'none' })
    const { result } = renderHook(() => useReaction(makePost({ MyReaction: 'liked' })), { wrapper: StatusMessageProvider })

    act(() => result.current.like())
    // Optimistic update applies before the mocked request resolves — a
    // naive implementation that always adds on "like" would show 4, not 2.
    expect(result.current.likeCount).toBe(2)
    expect(result.current.myReaction).toBe('none')

    await waitFor(() => expect(result.current.pending).toBe(false))
    expect(reactToPost).toHaveBeenCalledWith(1, true)
    expect(result.current.likeCount).toBe(2)
    expect(result.current.myReaction).toBe('none')
  })

  it('toggles the reaction off when clicking Dislike again while already disliked', async () => {
    vi.mocked(reactToPost).mockResolvedValue({ likeCount: 3, dislikeCount: 1, myReaction: 'none' })
    const { result } = renderHook(() => useReaction(makePost({ MyReaction: 'disliked' })), { wrapper: StatusMessageProvider })

    act(() => result.current.dislike())
    expect(result.current.dislikeCount).toBe(1)
    expect(result.current.myReaction).toBe('none')

    await waitFor(() => expect(result.current.pending).toBe(false))
    expect(reactToPost).toHaveBeenCalledWith(1, false)
    expect(result.current.dislikeCount).toBe(1)
    expect(result.current.myReaction).toBe('none')
  })
})

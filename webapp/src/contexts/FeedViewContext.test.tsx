import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { FeedViewProvider, useFeedView } from './FeedViewContext'

describe('FeedViewContext', () => {
  it('toggling a category from the "all" view starts a single-category filter', () => {
    const { result } = renderHook(() => useFeedView(), { wrapper: FeedViewProvider })
    act(() => result.current.toggleCategory('Sports'))
    expect(result.current.view).toEqual({ type: 'category', names: ['Sports'] })
  })

  it('toggling a second category adds it to the filter', () => {
    const { result } = renderHook(() => useFeedView(), { wrapper: FeedViewProvider })
    act(() => result.current.toggleCategory('Sports'))
    act(() => result.current.toggleCategory('Music'))
    expect(result.current.view).toEqual({ type: 'category', names: ['Sports', 'Music'] })
  })

  it('toggling an already-selected category removes it', () => {
    const { result } = renderHook(() => useFeedView(), { wrapper: FeedViewProvider })
    act(() => result.current.toggleCategory('Sports'))
    act(() => result.current.toggleCategory('Music'))
    act(() => result.current.toggleCategory('Sports'))
    expect(result.current.view).toEqual({ type: 'category', names: ['Music'] })
  })

  it('removing the last selected category falls back to the "all" view', () => {
    const { result } = renderHook(() => useFeedView(), { wrapper: FeedViewProvider })
    act(() => result.current.toggleCategory('Sports'))
    act(() => result.current.toggleCategory('Sports'))
    expect(result.current.view).toEqual({ type: 'all' })
  })

  it('toggling a category from a search view replaces it with a category filter', () => {
    const { result } = renderHook(() => useFeedView(), { wrapper: FeedViewProvider })
    act(() => result.current.showSearch('sushi'))
    act(() => result.current.toggleCategory('Cuisine'))
    expect(result.current.view).toEqual({ type: 'category', names: ['Cuisine'] })
  })

  it('showAllPosts resets to the "all" view regardless of prior selection', () => {
    const { result } = renderHook(() => useFeedView(), { wrapper: FeedViewProvider })
    act(() => result.current.toggleCategory('Sports'))
    act(() => result.current.toggleCategory('Music'))
    act(() => result.current.showAllPosts())
    expect(result.current.view).toEqual({ type: 'all' })
  })
})

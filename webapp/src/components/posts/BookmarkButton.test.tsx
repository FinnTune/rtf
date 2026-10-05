import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import type { Post } from '../../types'
import { StatusBanner } from '../common/StatusBanner'
import { BookmarkButton } from './BookmarkButton'

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

function renderButton(post: Post) {
  return render(
    <StatusMessageProvider>
      <StatusBanner />
      <BookmarkButton post={post} />
    </StatusMessageProvider>,
  )
}

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('BookmarkButton', () => {
  it('shows "Bookmark" and no active state when not yet bookmarked', () => {
    renderButton(makePost({ MyBookmark: false }))
    const button = screen.getByRole('button', { name: 'Bookmark' })
    expect(button).not.toHaveClass('active')
    expect(button).toHaveAttribute('aria-pressed', 'false')
  })

  it('shows "Bookmarked" and an active state when already bookmarked', () => {
    renderButton(makePost({ MyBookmark: true }))
    const button = screen.getByRole('button', { name: 'Bookmarked' })
    expect(button).toHaveClass('active')
    expect(button).toHaveAttribute('aria-pressed', 'true')
  })

  it('sends the correct request and flips to "Bookmarked" on click', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ bookmarked: true }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    renderButton(makePost({ MyBookmark: false }))

    await userEvent.click(screen.getByRole('button', { name: 'Bookmark' }))
    expect(await screen.findByRole('button', { name: 'Bookmarked' })).toHaveClass('active')

    const call = fetchMock.mock.calls.find(([input]) => requestUrl(input as string | URL | Request).startsWith('/bookmarkPost'))
    expect(call).toBeDefined()
    const body = JSON.parse((call![1] as RequestInit).body as string)
    expect(body).toEqual({ post_id: 1 })
  })

  it('rolls back and shows an error if the request fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('Failed to update bookmark', { status: 500 })))
    renderButton(makePost({ MyBookmark: false }))

    await userEvent.click(screen.getByRole('button', { name: 'Bookmark' }))
    expect(await screen.findByText('Err: Failed to update bookmark')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Bookmark' })).not.toHaveClass('active'))
  })
})

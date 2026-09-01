import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import type { Post } from '../../types'
import { StatusBanner } from '../common/StatusBanner'
import { ReactionButtons } from './ReactionButtons'

function makePost(overrides: Partial<Post> = {}): Post {
  return {
    PostId: 1,
    UserId: 1,
    Title: 'Test',
    Content: 'Body',
    Author: 'alice',
    Created: '2026-01-01',
    ImgURL: '',
    LikeCount: 2,
    DislikeCount: 1,
    MyReaction: 'none',
    ...overrides,
  }
}

function renderButtons(post: Post) {
  return render(
    <StatusMessageProvider>
      <StatusBanner />
      <ReactionButtons post={post} />
    </StatusMessageProvider>,
  )
}

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ReactionButtons', () => {
  it('shows the initial counts and no active state when there is no reaction yet', () => {
    renderButtons(makePost())
    expect(screen.getByRole('button', { name: 'Like (2)' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Dislike (1)' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Like (2)' })).not.toHaveClass('active')
  })

  it('marks the existing reaction as active on mount', () => {
    renderButtons(makePost({ MyReaction: 'liked' }))
    expect(screen.getByRole('button', { name: 'Like (2)' })).toHaveClass('active')
  })

  it('optimistically updates the count and active state immediately on click, before the server responds', async () => {
    let resolveFetch: (value: Response) => void = () => {}
    vi.stubGlobal(
      'fetch',
      vi.fn(() => new Promise<Response>((resolve) => (resolveFetch = resolve))),
    )
    renderButtons(makePost())

    await userEvent.click(screen.getByRole('button', { name: 'Like (2)' }))
    // The server hasn't responded yet, but the click should already read as optimistic +1 and active.
    expect(screen.getByRole('button', { name: 'Like (3)' })).toHaveClass('active')

    resolveFetch(new Response(JSON.stringify({ like_count: 3, dislike_count: 1, my_reaction: 'liked' }), { status: 200 }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Like (3)' })).toBeInTheDocument())
  })

  it('sends the correct request and reconciles with the server response', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ like_count: 5, dislike_count: 1, my_reaction: 'liked' }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    renderButtons(makePost())

    await userEvent.click(screen.getByRole('button', { name: 'Like (2)' }))
    expect(await screen.findByRole('button', { name: 'Like (5)' })).toHaveClass('active')

    const call = fetchMock.mock.calls.find(([input]) => requestUrl(input as string | URL | Request).startsWith('/reactToPost'))
    expect(call).toBeDefined()
    const body = JSON.parse((call![1] as RequestInit).body as string)
    expect(body).toEqual({ post_id: 1, is_liked: true })
  })

  it('rolls back to the previous state and shows an error if the request fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('Failed to react to post', { status: 500 })))
    renderButtons(makePost())

    await userEvent.click(screen.getByRole('button', { name: 'Like (2)' }))
    expect(await screen.findByText('Err: Failed to react to post')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Like (2)' })).not.toHaveClass('active')
  })

  it('clicking dislike while already liked switches the reaction', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ like_count: 1, dislike_count: 2, my_reaction: 'disliked' }), { status: 200 })))
    renderButtons(makePost({ LikeCount: 2, DislikeCount: 1, MyReaction: 'liked' }))

    await userEvent.click(screen.getByRole('button', { name: /^Dislike/ }))
    expect(await screen.findByRole('button', { name: 'Dislike (2)' })).toHaveClass('active')
    expect(screen.getByRole('button', { name: 'Like (1)' })).not.toHaveClass('active')
  })
})

import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { WebSocketProvider } from '../../contexts/WebSocketContext'
import { ControllableFakeWebSocket, checkLoginResponse, requestUrl } from '../../testUtils/chatTestHarness'
import type { Comment } from '../../types'
import { CommentList } from './CommentList'

function makeComment(overrides: Partial<Comment> = {}): Comment {
  return {
    id: 1,
    user_id: 1,
    post_id: 5,
    username: 'alice',
    content: 'a comment',
    created_at: '2026-01-01',
    ...overrides,
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('CommentList', () => {
  it('removes a comment when a comment-deleted broadcast arrives for this post, but ignores one for a different post', async () => {
    ControllableFakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    const comments = [makeComment({ id: 1, content: 'first comment' }), makeComment({ id: 2, content: 'second comment' })]
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/comments')) {
          return new Response(JSON.stringify(comments), { status: 200, headers: { 'X-Total-Count': '2' } })
        }
        throw new Error('Unexpected fetch: ' + url)
      }),
    )

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <WebSocketProvider>
            <CommentList postId={5} />
          </WebSocketProvider>
        </AuthProvider>
      </StatusMessageProvider>,
    )

    expect(await screen.findByText(/first comment/)).toBeInTheDocument()
    expect(screen.getByText(/second comment/)).toBeInTheDocument()

    await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
    const socket = ControllableFakeWebSocket.instances[0]
    act(() => socket.simulateOpen())

    // A broadcast for a different post must be ignored.
    act(() => socket.simulateMessage('comment-deleted', { post_id: 999, comment_id: 1 }))
    expect(screen.getByText(/first comment/)).toBeInTheDocument()

    act(() => socket.simulateMessage('comment-deleted', { post_id: 5, comment_id: 1 }))
    await waitFor(() => expect(screen.queryByText(/first comment/)).not.toBeInTheDocument())
    expect(screen.getByText(/second comment/)).toBeInTheDocument()
  })

  it('updates a comment content when a comment-edited broadcast arrives for this post, but ignores one for a different post', async () => {
    ControllableFakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    const comments = [makeComment({ id: 1, content: 'original content' })]
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/comments')) {
          return new Response(JSON.stringify(comments), { status: 200, headers: { 'X-Total-Count': '1' } })
        }
        throw new Error('Unexpected fetch: ' + url)
      }),
    )

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <WebSocketProvider>
            <CommentList postId={5} />
          </WebSocketProvider>
        </AuthProvider>
      </StatusMessageProvider>,
    )

    expect(await screen.findByText(/original content/)).toBeInTheDocument()

    await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
    const socket = ControllableFakeWebSocket.instances[0]
    act(() => socket.simulateOpen())

    // A broadcast for a different post must be ignored.
    act(() => socket.simulateMessage('comment-edited', { post_id: 999, comment_id: 1, content: 'wrong post' }))
    expect(screen.getByText(/original content/)).toBeInTheDocument()

    act(() => socket.simulateMessage('comment-edited', { post_id: 5, comment_id: 1, content: 'edited content' }))
    await waitFor(() => expect(screen.getByText(/edited content/)).toBeInTheDocument())
    expect(screen.queryByText(/original content/)).not.toBeInTheDocument()
  })

  it('appends a new comment when a comment-added broadcast arrives for this post, but ignores one for a different post', async () => {
    ControllableFakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    const comments = [makeComment({ id: 1, content: 'existing comment' })]
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/comments')) {
          return new Response(JSON.stringify(comments), { status: 200, headers: { 'X-Total-Count': '1' } })
        }
        throw new Error('Unexpected fetch: ' + url)
      }),
    )

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <WebSocketProvider>
            <CommentList postId={5} />
          </WebSocketProvider>
        </AuthProvider>
      </StatusMessageProvider>,
    )

    expect(await screen.findByText(/existing comment/)).toBeInTheDocument()

    await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
    const socket = ControllableFakeWebSocket.instances[0]
    act(() => socket.simulateOpen())

    // A broadcast for a different post must be ignored.
    act(() =>
      socket.simulateMessage('comment-added', {
        id: 2,
        user_id: 9,
        post_id: 999,
        username: 'bob',
        content: 'wrong post comment',
        created_at: '2026-01-02',
      }),
    )
    expect(screen.queryByText(/wrong post comment/)).not.toBeInTheDocument()

    act(() =>
      socket.simulateMessage('comment-added', {
        id: 3,
        user_id: 9,
        post_id: 5,
        username: 'bob',
        content: 'new comment from bob',
        created_at: '2026-01-02',
      }),
    )
    await waitFor(() => expect(screen.getByText(/new comment from bob/)).toBeInTheDocument())
    expect(screen.getByText(/existing comment/)).toBeInTheDocument()
  })

  // Regression test for a bug where a comment-added broadcast, received
  // while the viewer had only loaded part of a post's comment history,
  // was appended immediately — then re-fetched a second time (duplicated,
  // and out of chronological order) once the viewer clicked "Load more",
  // because the offset-based fetch that triggers re-covers the same
  // range the live comment had already been spliced into.
  it('does not duplicate a comment-added broadcast when the viewer has not loaded the full comment history yet', async () => {
    ControllableFakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    const firstComment = makeComment({ id: 1, content: 'first comment' })
    const remainingComments = [
      makeComment({ id: 2, content: 'second comment' }),
      makeComment({ id: 3, content: 'third comment' }),
      makeComment({ id: 4, content: 'broadcast comment' }),
    ]
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/comments')) {
          const offset = new URL(url, 'https://localhost').searchParams.get('offset')
          if (offset === '0') {
            // Only the first comment "loaded" so far — 3 more remain,
            // including the one about to arrive live via broadcast (id 4).
            return new Response(JSON.stringify([firstComment]), { status: 200, headers: { 'X-Total-Count': '3' } })
          }
          // "Load more" re-fetches everything from offset 1 onward. By
          // the time this runs, the broadcast below has bumped the
          // server-reported total to 4 — comment 4 genuinely belongs in
          // this range now, oldest-first, same as the real backend would
          // return it.
          return new Response(JSON.stringify(remainingComments), { status: 200, headers: { 'X-Total-Count': '4' } })
        }
        throw new Error('Unexpected fetch: ' + url)
      }),
    )

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <WebSocketProvider>
            <CommentList postId={5} />
          </WebSocketProvider>
        </AuthProvider>
      </StatusMessageProvider>,
    )

    expect(await screen.findByText(/first comment/)).toBeInTheDocument()
    const loadMoreButton = await screen.findByRole('button', { name: 'Load more comments' })

    await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
    const socket = ControllableFakeWebSocket.instances[0]
    act(() => socket.simulateOpen())

    // Must NOT be appended directly — it would render ahead of comments 2
    // and 3, which this viewer hasn't loaded yet.
    act(() =>
      socket.simulateMessage('comment-added', {
        id: 4,
        user_id: 9,
        post_id: 5,
        username: 'bob',
        content: 'broadcast comment',
        created_at: '2026-01-02',
      }),
    )
    expect(screen.queryByText(/broadcast comment/)).not.toBeInTheDocument()

    await userEvent.click(loadMoreButton)

    await waitFor(() => expect(screen.getByText(/broadcast comment/)).toBeInTheDocument())
    expect(screen.getAllByText(/broadcast comment/)).toHaveLength(1)
    expect(screen.getByText(/second comment/)).toBeInTheDocument()
    expect(screen.getByText(/third comment/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Load more comments' })).not.toBeInTheDocument()
  })
})

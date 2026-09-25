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

  // Regression test for a bug where deleting a comment (locally, via the
  // Delete button) removed it from the list but left offset unchanged —
  // so the next "Load more" fetch, still anchored at the pre-delete
  // offset, silently skipped over the comment that had shifted into that
  // now-vacated slot, permanently hiding it.
  it('does not skip a comment on "Load more" after locally deleting an own comment', async () => {
    ControllableFakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const firstPage = [makeComment({ id: 1, content: 'first comment' }), makeComment({ id: 2, content: 'second comment' })]
    const secondPage = [makeComment({ id: 3, content: 'third comment' })]
    const requestedOffsets: (string | null)[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/deleteComment')) return new Response(null, { status: 200 })
        if (url.startsWith('/comments')) {
          const offset = new URL(url, 'https://localhost').searchParams.get('offset')
          requestedOffsets.push(offset)
          if (offset === '0') {
            return new Response(JSON.stringify(firstPage), { status: 200, headers: { 'X-Total-Count': '3' } })
          }
          return new Response(JSON.stringify(secondPage), { status: 200, headers: { 'X-Total-Count': '2' } })
        }
        throw new Error('Unexpected fetch: ' + (init?.method ?? 'GET') + ' ' + url)
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
    const deleteButtons = await screen.findAllByRole('button', { name: 'Delete' })
    await userEvent.click(deleteButtons[0])
    await waitFor(() => expect(screen.queryByText(/first comment/)).not.toBeInTheDocument())

    const loadMoreButton = await screen.findByRole('button', { name: 'Load more comments' })
    await userEvent.click(loadMoreButton)

    await waitFor(() => expect(screen.getByText(/third comment/)).toBeInTheDocument())
    expect(screen.getByText(/second comment/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Load more comments' })).not.toBeInTheDocument()
    // The "Load more" fetch must have been anchored at offset 1 (shrunk
    // from 2 by the delete above), not stayed at 2.
    expect(requestedOffsets).toEqual(['0', '1'])
  })

  // Regression test for the same offset-skip bug as above, but via a
  // comment-deleted broadcast for a comment this viewer HAD already
  // loaded — the handler must decrement offset just like the local
  // delete path does.
  it('does not skip a comment on "Load more" after a comment-deleted broadcast for an already-loaded comment', async () => {
    ControllableFakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    const firstPage = [makeComment({ id: 1, content: 'first comment' }), makeComment({ id: 2, content: 'second comment' })]
    const secondPage = [makeComment({ id: 3, content: 'third comment' })]
    const requestedOffsets: (string | null)[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/comments')) {
          const offset = new URL(url, 'https://localhost').searchParams.get('offset')
          requestedOffsets.push(offset)
          if (offset === '0') {
            return new Response(JSON.stringify(firstPage), { status: 200, headers: { 'X-Total-Count': '3' } })
          }
          return new Response(JSON.stringify(secondPage), { status: 200, headers: { 'X-Total-Count': '2' } })
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
    expect(await screen.findByText(/second comment/)).toBeInTheDocument()

    await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
    const socket = ControllableFakeWebSocket.instances[0]
    act(() => socket.simulateOpen())

    // Comment 1 is part of the already-loaded prefix.
    act(() => socket.simulateMessage('comment-deleted', { post_id: 5, comment_id: 1 }))
    await waitFor(() => expect(screen.queryByText(/first comment/)).not.toBeInTheDocument())

    const loadMoreButton = await screen.findByRole('button', { name: 'Load more comments' })
    await userEvent.click(loadMoreButton)

    await waitFor(() => expect(screen.getByText(/third comment/)).toBeInTheDocument())
    expect(screen.getByText(/second comment/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Load more comments' })).not.toBeInTheDocument()
    // The "Load more" fetch must have been anchored at offset 1 (shrunk
    // from 2 by the broadcast delete above), not stayed at 2.
    expect(requestedOffsets).toEqual(['0', '1'])
  })

  // Sanity check for the other direction: a comment-deleted broadcast for
  // a comment NOT yet loaded (still in the unfetched remainder past
  // offset) must leave offset untouched — only total should shrink. This
  // case already worked before the fix above; confirms it still does.
  // (If offset were wrongly decremented here too, the "Load more" fetch
  // below would re-request from offset 1 instead of 2, re-fetching
  // comment 2 a second time.)
  it('does not change offset when a comment-deleted broadcast targets a comment that was not yet loaded', async () => {
    ControllableFakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    const firstPage = [makeComment({ id: 1, content: 'first comment' }), makeComment({ id: 2, content: 'second comment' })]
    const secondPage = [makeComment({ id: 4, content: 'fourth comment' })]
    const requestedOffsets: (string | null)[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/comments')) {
          const offset = new URL(url, 'https://localhost').searchParams.get('offset')
          requestedOffsets.push(offset)
          if (offset === '0') {
            // 2 loaded, 2 more (ids 3 and 4) remain unloaded.
            return new Response(JSON.stringify(firstPage), { status: 200, headers: { 'X-Total-Count': '4' } })
          }
          return new Response(JSON.stringify(secondPage), { status: 200, headers: { 'X-Total-Count': '3' } })
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
    expect(await screen.findByText(/second comment/)).toBeInTheDocument()

    await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
    const socket = ControllableFakeWebSocket.instances[0]
    act(() => socket.simulateOpen())

    // Comment 3 was never loaded (only 1 and 2 were).
    act(() => socket.simulateMessage('comment-deleted', { post_id: 5, comment_id: 3 }))

    const loadMoreButton = await screen.findByRole('button', { name: 'Load more comments' })
    await userEvent.click(loadMoreButton)

    await waitFor(() => expect(screen.getByText(/fourth comment/)).toBeInTheDocument())
    expect(screen.getAllByText(/second comment/)).toHaveLength(1)
    expect(screen.queryByRole('button', { name: 'Load more comments' })).not.toBeInTheDocument()
    // The "Load more" fetch must have stayed anchored at offset 2 — the
    // deleted comment was never loaded, so offset must not have shrunk.
    expect(requestedOffsets).toEqual(['0', '2'])
  })
})

import { act, render, screen, waitFor } from '@testing-library/react'
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
})

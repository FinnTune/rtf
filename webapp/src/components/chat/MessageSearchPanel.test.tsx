import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ControllableFakeWebSocket, chatWrapper as wrapper, checkLoginResponse, requestUrl } from '../../testUtils/chatTestHarness'
import { MessageSearchPanel } from './MessageSearchPanel'

async function setup(searchResponse: unknown) {
  ControllableFakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
      if (url.startsWith('/searchMessages')) return new Response(JSON.stringify(searchResponse), { status: 200 })
      throw new Error('Unexpected fetch in test: ' + url)
    }),
  )

  const view = render(<MessageSearchPanel />, { wrapper })
  await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  const socket = ControllableFakeWebSocket.instances[0]
  socket.simulateOpen()
  return { socket, ...view }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('MessageSearchPanel', () => {
  it('searches and renders matching results with sender and conversation title', async () => {
    const { socket } = await setup([{ id: 1, conversation_id: 5, from: 'bob', message: 'hi there', created_at: '2026-01-01T00:00:00Z' }])

    // conversations-list gives the panel a title to resolve conversation_id 5 to.
    socket.simulateMessage('conversations-list', [
      {
        conversation_id: 5,
        is_group: false,
        members: [
          { user_id: 1, username: 'alice' },
          { user_id: 2, username: 'bob' },
        ],
        read_states: [],
      },
    ])

    await userEvent.type(screen.getByPlaceholderText('Search your messages...'), 'hi there')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(await screen.findByText('hi there')).toBeInTheDocument()
    expect(screen.getByText('bob')).toBeInTheDocument()
    expect(screen.getByText('in bob', { exact: false })).toBeInTheDocument()
  })

  it('shows an empty state when there are no matches', async () => {
    await setup([])

    await userEvent.type(screen.getByPlaceholderText('Search your messages...'), 'nothing')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(await screen.findByText('No messages found.')).toBeInTheDocument()
  })

  it('clicking a result opens that conversation', async () => {
    const { socket } = await setup([{ id: 1, conversation_id: 5, from: 'bob', message: 'hi there', created_at: '2026-01-01T00:00:00Z' }])

    socket.simulateMessage('conversations-list', [
      {
        conversation_id: 5,
        is_group: false,
        members: [
          { user_id: 1, username: 'alice' },
          { user_id: 2, username: 'bob' },
        ],
        read_states: [],
      },
    ])

    await userEvent.type(screen.getByPlaceholderText('Search your messages...'), 'hi there')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))
    await userEvent.click(await screen.findByText('hi there'))

    // Opening an already-known conversation goes straight to fetching
    // history — no open-direct-chat round trip needed.
    await waitFor(() => {
      const historyRequest = socket.sent.find((frame) => JSON.parse(frame).type === 'get-chat-history')
      expect(historyRequest).toBeDefined()
      expect(JSON.parse(historyRequest!).payload).toEqual({ conversation_id: 5, offset: 0, limit: 10 })
    })
  })
})

import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from './AuthContext'
import { ChatProvider, useChat } from './ChatContext'
import { StatusMessageProvider } from './StatusMessageContext'
import { WebSocketProvider } from './WebSocketContext'

class ControllableFakeWebSocket {
  static instances: ControllableFakeWebSocket[] = []
  static readonly OPEN = 1
  readyState = 0
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onmessage: ((event: MessageEvent<string>) => void) | null = null
  sent: string[] = []
  url: string
  constructor(url: string) {
    this.url = url
    ControllableFakeWebSocket.instances.push(this)
  }
  send(data: string) {
    this.sent.push(data)
  }
  close() {
    this.onclose?.()
  }
  simulateOpen() {
    this.readyState = 1
    this.onopen?.()
  }
  simulateMessage(type: string, payload: unknown) {
    this.onmessage?.({ data: JSON.stringify({ type, payload }) } as MessageEvent<string>)
  }
}

function checkLoginResponse(username: string) {
  return new Response(
    JSON.stringify({
      loggedIn: true,
      id: 1,
      username,
      email: 'a@example.com',
      joined: '2026-01-01',
      otp: 'otp-' + Math.random(),
    }),
    { status: 200 },
  )
}

function wrapper({ children }: { children: ReactNode }) {
  return (
    <StatusMessageProvider>
      <AuthProvider>
        <WebSocketProvider>
          <ChatProvider>{children}</ChatProvider>
        </WebSocketProvider>
      </AuthProvider>
    </StatusMessageProvider>
  )
}

async function setup(myUsername = 'alice') {
  ControllableFakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  // AuthProvider's mount effect and WebSocketProvider's fresh-OTP mint both
  // call /checkLogin — a Response body can only be read once, so a fresh
  // Response is needed per call, not the same one reused.
  vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(checkLoginResponse(myUsername))))

  const { result } = renderHook(() => useChat(), { wrapper })
  await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  const socket = ControllableFakeWebSocket.instances[0]
  act(() => socket.simulateOpen())
  return { result, socket }
}

// Drives the open-direct-chat request/chat-opened response round trip and
// returns the resulting conversation id.
function openBobConversation(socket: ControllableFakeWebSocket, result: { current: ReturnType<typeof useChat> }, conversationId = 5) {
  act(() => result.current.openDirectChat('bob'))
  act(() =>
    socket.simulateMessage('chat-opened', {
      conversation_id: conversationId,
      is_group: false,
      members: [
        { user_id: 1, username: 'alice' },
        { user_id: 2, username: 'bob' },
      ],
    }),
  )
  return conversationId
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ChatContext', () => {
  it('updates onlineUsers from a users-online event', async () => {
    const { result, socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true, bob: true }))
    await waitFor(() => expect(result.current.onlineUsers).toEqual(['alice', 'bob']))
  })

  it('marks a sender unread when their message arrives with no window open', async () => {
    const { result, socket } = await setup()
    act(() => socket.simulateMessage('sent-message', { conversation_id: 5, from: 'bob', message: 'hi', sent: '2026-01-01T00:00:00Z' }))
    await waitFor(() => expect(result.current.unreadUsernames.has('bob')).toBe(true))
  })

  it('appends to the open window instead of marking unread when the window is already open', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() => socket.simulateMessage('sent-message', { conversation_id: convId, from: 'bob', message: 'hi', sent: '2026-01-01T00:00:00Z' }))
    await waitFor(() => expect(result.current.openWindows[convId].messages).toHaveLength(1))
    expect(result.current.unreadUsernames.has('bob')).toBe(false)
  })

  it('openDirectChat resolves an unknown user via open-direct-chat and requests history for page 1', async () => {
    const { result, socket } = await setup()

    act(() => result.current.openDirectChat('bob'))
    const openRequest = JSON.parse(socket.sent[socket.sent.length - 1]) as { type: string; payload: unknown }
    expect(openRequest).toEqual({ type: 'open-direct-chat', payload: { username: 'bob' } })

    act(() =>
      socket.simulateMessage('chat-opened', {
        conversation_id: 5,
        is_group: false,
        members: [
          { user_id: 1, username: 'alice' },
          { user_id: 2, username: 'bob' },
        ],
      }),
    )
    await waitFor(() => expect(result.current.openWindows[5]).toBeDefined())
    const historyRequest = JSON.parse(socket.sent[socket.sent.length - 1]) as { type: string; payload: unknown }
    expect(historyRequest).toEqual({ type: 'get-chat-history', payload: { conversation_id: 5, offset: 0, limit: 10 } })
  })

  it('openDirectChat on an already-known conversation skips the open-direct-chat round trip', async () => {
    const { result, socket } = await setup()
    // Learn about bob's conversation via an incoming message, without ever
    // opening a window for it.
    act(() => socket.simulateMessage('sent-message', { conversation_id: 5, from: 'bob', message: 'hi', sent: '2026-01-01T00:00:00Z' }))
    await waitFor(() => expect(result.current.unreadUsernames.has('bob')).toBe(true))

    act(() => result.current.openDirectChat('bob'))
    await waitFor(() => expect(result.current.unreadUsernames.has('bob')).toBe(false))
    const lastFrame = JSON.parse(socket.sent[socket.sent.length - 1]) as { type: string; payload: unknown }
    // Goes straight to fetching history — no open-direct-chat round trip,
    // since the conversation is already known.
    expect(lastFrame).toEqual({ type: 'get-chat-history', payload: { conversation_id: 5, offset: 0, limit: 10 } })
  })

  it('reopening an already-open window does not re-request history', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())
    const callsBefore = socket.sent.length

    act(() => result.current.openDirectChat('bob'))
    expect(socket.sent).toHaveLength(callsBefore)
  })

  it('routes a chat_history batch to the window awaiting it', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() =>
      socket.simulateMessage('chat_history', [
        { from: 'bob', message: 'first', created_at: '2026-01-01T00:00:00Z' },
        { from: 'alice', message: 'second', created_at: '2026-01-01T00:01:00Z' },
      ]),
    )
    await waitFor(() => expect(result.current.openWindows[convId].messages).toHaveLength(2))
    expect(result.current.openWindows[convId].messages.map((m) => m.message)).toEqual(['first', 'second'])
    expect(result.current.openWindows[convId].loadingHistory).toBe(false)
  })

  it('sendMessage appends the message locally, since the server never echoes it back to the sender', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() => result.current.sendMessage(convId, 'hello there'))
    expect(result.current.openWindows[convId].messages).toHaveLength(1)
    expect(result.current.openWindows[convId].messages[0]).toMatchObject({ from: 'alice', message: 'hello there' })
    const sentFrame = JSON.parse(socket.sent[socket.sent.length - 1]) as { type: string; payload: unknown }
    expect(sentFrame).toEqual({ type: 'new-message', payload: { conversation_id: convId, message: 'hello there' } })
  })

  it('closeChat removes the window', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() => result.current.closeChat(convId))
    expect(result.current.openWindows[convId]).toBeUndefined()
  })

  it('a pushed chat-opened event (e.g. being added to a group) opens a window automatically', async () => {
    const { result, socket } = await setup()

    act(() =>
      socket.simulateMessage('chat-opened', {
        conversation_id: 9,
        is_group: true,
        name: 'Trip Planning',
        members: [
          { user_id: 1, username: 'alice' },
          { user_id: 2, username: 'bob' },
          { user_id: 3, username: 'carol' },
        ],
      }),
    )

    await waitFor(() => expect(result.current.openWindows[9]).toBeDefined())
    expect(result.current.openWindows[9].isGroup).toBe(true)
    expect(result.current.openWindows[9].title).toBe('Trip Planning')
    expect(result.current.groupChats.map((g) => g.conversation_id)).toContain(9)
  })
})

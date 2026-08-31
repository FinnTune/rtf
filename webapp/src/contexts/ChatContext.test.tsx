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
    act(() => socket.simulateMessage('sent-message', { from: 'bob', to: 'alice', message: 'hi', sent: '2026-01-01T00:00:00Z' }))
    await waitFor(() => expect(result.current.unreadUsers.has('bob')).toBe(true))
  })

  it('appends to the open window instead of marking unread when the window is already open', async () => {
    const { result, socket } = await setup()
    act(() => result.current.openChat('bob'))
    await waitFor(() => expect(result.current.openWindows.bob).toBeDefined())

    act(() => socket.simulateMessage('sent-message', { from: 'bob', to: 'alice', message: 'hi', sent: '2026-01-01T00:00:00Z' }))
    await waitFor(() => expect(result.current.openWindows.bob.messages).toHaveLength(1))
    expect(result.current.unreadUsers.has('bob')).toBe(false)
  })

  it('openChat clears any existing unread flag and requests history for page 1', async () => {
    const { result, socket } = await setup()
    act(() => socket.simulateMessage('sent-message', { from: 'bob', to: 'alice', message: 'hi', sent: '2026-01-01T00:00:00Z' }))
    await waitFor(() => expect(result.current.unreadUsers.has('bob')).toBe(true))

    act(() => result.current.openChat('bob'))
    await waitFor(() => expect(result.current.unreadUsers.has('bob')).toBe(false))
    const historyRequest = JSON.parse(socket.sent[socket.sent.length - 1])
    expect(historyRequest).toEqual({ type: 'get-chat-history', payload: { from: 'alice', to: 'bob', offset: 0, limit: 10 } })
  })

  it('reopening an already-open window does not re-request history', async () => {
    const { result, socket } = await setup()
    act(() => result.current.openChat('bob'))
    await waitFor(() => expect(result.current.openWindows.bob).toBeDefined())
    const callsBefore = socket.sent.length

    act(() => result.current.openChat('bob'))
    expect(socket.sent).toHaveLength(callsBefore)
  })

  it("routes a chat_history batch to the right window using the first message's other party", async () => {
    const { result, socket } = await setup()
    act(() => result.current.openChat('bob'))
    await waitFor(() => expect(result.current.openWindows.bob).toBeDefined())

    act(() =>
      socket.simulateMessage('chat_history', [
        { from: 'bob', to: 'alice', message: 'first', created_at: '2026-01-01T00:00:00Z' },
        { from: 'alice', to: 'bob', message: 'second', created_at: '2026-01-01T00:01:00Z' },
      ]),
    )
    await waitFor(() => expect(result.current.openWindows.bob.messages).toHaveLength(2))
    expect(result.current.openWindows.bob.messages.map((m) => m.message)).toEqual(['first', 'second'])
  })

  it('sendMessage appends the message locally, since the server never echoes it back to the sender', async () => {
    const { result, socket } = await setup()
    act(() => result.current.openChat('bob'))
    await waitFor(() => expect(result.current.openWindows.bob).toBeDefined())

    act(() => result.current.sendMessage('bob', 'hello there'))
    expect(result.current.openWindows.bob.messages).toHaveLength(1)
    expect(result.current.openWindows.bob.messages[0]).toMatchObject({ from: 'alice', to: 'bob', message: 'hello there' })
    const sentFrame = JSON.parse(socket.sent[socket.sent.length - 1])
    expect(sentFrame.type).toBe('new-message')
  })

  it('closeChat removes the window', async () => {
    const { result } = await setup()
    act(() => result.current.openChat('bob'))
    await waitFor(() => expect(result.current.openWindows.bob).toBeDefined())

    act(() => result.current.closeChat('bob'))
    expect(result.current.openWindows.bob).toBeUndefined()
  })
})

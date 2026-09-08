import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useChat } from './ChatContext'
import { useStatusMessage } from './StatusMessageContext'
import { ControllableFakeWebSocket, chatWrapper as wrapper, checkLoginResponse } from '../testUtils/chatTestHarness'

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

// Like setup(), but also exposes the status banner state — for tests
// asserting on the in-app toast a chat notification triggers.
async function setupWithStatus(myUsername = 'alice') {
  ControllableFakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(checkLoginResponse(myUsername))))

  const { result } = renderHook(() => ({ chat: useChat(), status: useStatusMessage() }), { wrapper })
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
      read_states: [
        { user_id: 1, username: 'alice', last_read_message_id: 0 },
        { user_id: 2, username: 'bob', last_read_message_id: 0 },
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
        read_states: [
          { user_id: 1, username: 'alice', last_read_message_id: 0 },
          { user_id: 2, username: 'bob', last_read_message_id: 0 },
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

  it('routes a chat_history batch to the window awaiting it and marks it read up to the newest message', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() =>
      socket.simulateMessage('chat_history', [
        { id: 10, from: 'bob', message: 'first', created_at: '2026-01-01T00:00:00Z' },
        { id: 11, from: 'alice', message: 'second', created_at: '2026-01-01T00:01:00Z' },
      ]),
    )
    await waitFor(() => expect(result.current.openWindows[convId].messages).toHaveLength(2))
    expect(result.current.openWindows[convId].messages.map((m) => m.message)).toEqual(['first', 'second'])
    expect(result.current.openWindows[convId].loadingHistory).toBe(false)

    const markReadFrame = JSON.parse(socket.sent[socket.sent.length - 1]) as { type: string; payload: unknown }
    expect(markReadFrame).toEqual({ type: 'mark-read', payload: { conversation_id: convId, message_id: 11 } })
  })

  it('a read-receipt event updates the window\'s read state for that member', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() => socket.simulateMessage('read-receipt', { conversation_id: convId, user_id: 2, username: 'bob', message_id: 7 }))
    await waitFor(() => expect(result.current.openWindows[convId].readStates.bob).toBe(7))
  })

  it('an incoming message while the window is open triggers mark-read for that message', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() =>
      socket.simulateMessage('sent-message', { id: 42, conversation_id: convId, from: 'bob', message: 'hi', sent: '2026-01-01T00:00:00Z' }),
    )
    await waitFor(() => expect(result.current.openWindows[convId].messages).toHaveLength(1))

    const markReadFrame = JSON.parse(socket.sent[socket.sent.length - 1]) as { type: string; payload: unknown }
    expect(markReadFrame).toEqual({ type: 'mark-read', payload: { conversation_id: convId, message_id: 42 } })
  })

  it('sendMessage appends the message locally, since the server never echoes it back to the sender', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() => result.current.sendMessage(convId, 'hello there'))
    expect(result.current.openWindows[convId].messages).toHaveLength(1)
    // id: 0 marks it unconfirmed — the server never echoes a sent message
    // back to its own sender, so this client never learns its real id.
    // clientMsgId is the correlation token generated for this send (see
    // ChatContext.tsx's sendMessage) — asserted present, not pinned to a
    // specific value, which would just test the counter's implementation.
    expect(result.current.openWindows[convId].messages[0]).toMatchObject({ id: 0, from: 'alice', message: 'hello there' })
    const clientMsgId = result.current.openWindows[convId].messages[0].clientMsgId
    expect(clientMsgId).toEqual(expect.any(String))
    const sentFrame = JSON.parse(socket.sent[socket.sent.length - 1]) as {
      type: string
      payload: { conversation_id: number; message: string; client_msg_id: string }
    }
    expect(sentFrame).toEqual({
      type: 'new-message',
      payload: { conversation_id: convId, message: 'hello there', client_msg_id: clientMsgId },
    })
  })

  it('a message-ack reconciles the sender\'s own optimistic message with its real id', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() => result.current.sendMessage(convId, 'hello there'))
    expect(result.current.openWindows[convId].messages[0].id).toBe(0)
    const clientMsgId = result.current.openWindows[convId].messages[0].clientMsgId

    act(() => socket.simulateMessage('message-ack', { conversation_id: convId, id: 99, client_msg_id: clientMsgId }))
    await waitFor(() => expect(result.current.openWindows[convId].messages[0].id).toBe(99))
  })

  // Guards the actual bug this correlation-by-token design fixes: with two
  // sends outstanding at once, an ack must reconcile the specific local
  // echo it belongs to — matching "the oldest unconfirmed (id: 0) entry"
  // (the old approach) would attach the second ack to the first message
  // whenever they don't resolve in send order.
  it('two outstanding sends reconcile independently, even acked out of order', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    act(() => result.current.sendMessage(convId, 'first'))
    act(() => result.current.sendMessage(convId, 'second'))
    const [firstId, secondId] = result.current.openWindows[convId].messages.map((m) => m.clientMsgId)
    expect(firstId).not.toBe(secondId)

    // Acked out of order: the second send's ack arrives first.
    act(() => socket.simulateMessage('message-ack', { conversation_id: convId, id: 202, client_msg_id: secondId }))
    act(() => socket.simulateMessage('message-ack', { conversation_id: convId, id: 101, client_msg_id: firstId }))

    await waitFor(() => {
      const messages = result.current.openWindows[convId].messages
      expect(messages.find((m) => m.message === 'first')?.id).toBe(101)
      expect(messages.find((m) => m.message === 'second')?.id).toBe(202)
    })
  })

  it('a chat-error correlated to a send marks that specific message failed', async () => {
    const { result, socket } = await setupWithStatus()
    const convId = openBobConversation(socket, { current: result.current.chat })
    await waitFor(() => expect(result.current.chat.openWindows[convId]).toBeDefined())

    act(() => result.current.chat.sendMessage(convId, 'x'.repeat(1001)))
    const clientMsgId = result.current.chat.openWindows[convId].messages[0].clientMsgId

    act(() => socket.simulateMessage('chat-error', { message: 'message must be 1-1000 characters', client_msg_id: clientMsgId }))

    await waitFor(() => expect(result.current.chat.openWindows[convId].messages[0].failed).toBe(true))
    expect(result.current.status.text).toContain('message must be 1-1000 characters')
  })

  it('a send with no ack or chat-error within the timeout is marked failed', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    vi.useFakeTimers()
    act(() => result.current.sendMessage(convId, 'into the void'))
    expect(result.current.openWindows[convId].messages[0].failed).toBeUndefined()

    // Simulates the server's two genuinely silent drop paths (a stale
    // conversation_id the sender isn't a member of, or the per-connection
    // rate limit — see ws-manager.go) by simply never responding at all.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(15_000)
    })

    expect(result.current.openWindows[convId].messages[0].failed).toBe(true)
    vi.useRealTimers()
  })

  it('a timely ack cancels the failure timeout — no false positive on a normal round trip', async () => {
    const { result, socket } = await setup()
    const convId = openBobConversation(socket, result)
    await waitFor(() => expect(result.current.openWindows[convId]).toBeDefined())

    vi.useFakeTimers()
    act(() => result.current.sendMessage(convId, 'hi'))
    const clientMsgId = result.current.openWindows[convId].messages[0].clientMsgId

    act(() => socket.simulateMessage('message-ack', { conversation_id: convId, id: 7, client_msg_id: clientMsgId }))

    await act(async () => {
      await vi.advanceTimersByTimeAsync(15_000)
    })

    expect(result.current.openWindows[convId].messages[0].failed).toBeUndefined()
    expect(result.current.openWindows[convId].messages[0].id).toBe(7)
    vi.useRealTimers()
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
        read_states: [
          { user_id: 1, username: 'alice', last_read_message_id: 0 },
          { user_id: 2, username: 'bob', last_read_message_id: 0 },
          { user_id: 3, username: 'carol', last_read_message_id: 0 },
        ],
      }),
    )

    await waitFor(() => expect(result.current.openWindows[9]).toBeDefined())
    expect(result.current.openWindows[9].isGroup).toBe(true)
    expect(result.current.openWindows[9].title).toBe('Trip Planning')
    expect(result.current.groupChats.map((g) => g.conversation_id)).toContain(9)
  })

  it('shows an in-app toast when a message arrives for a conversation with no open window', async () => {
    const { result, socket } = await setupWithStatus()

    act(() =>
      socket.simulateMessage('sent-message', { id: 1, conversation_id: 5, from: 'bob', message: 'hi there', sent: '2026-01-01T00:00:00Z' }),
    )

    await waitFor(() => expect(result.current.status.text).toBe('New message from bob: hi there'))
    expect(result.current.status.type).toBe('info')
  })

  it('does not show an in-app toast when the message arrives for an already-open window', async () => {
    const { result, socket } = await setupWithStatus()
    const convId = openBobConversation(socket, { current: result.current.chat })
    await waitFor(() => expect(result.current.chat.openWindows[convId]).toBeDefined())

    act(() =>
      socket.simulateMessage('sent-message', { id: 1, conversation_id: convId, from: 'bob', message: 'hi there', sent: '2026-01-01T00:00:00Z' }),
    )

    await waitFor(() => expect(result.current.chat.openWindows[convId].messages).toHaveLength(1))
    expect(result.current.status.text).toBe('')
  })

  it('shows a toast for a chat-error event (e.g. an over-length message rejected server-side)', async () => {
    const { result, socket } = await setupWithStatus()

    act(() => socket.simulateMessage('chat-error', { message: 'message must be 1-1000 characters' }))

    await waitFor(() => expect(result.current.status.text).toBe('Err: message must be 1-1000 characters'))
    expect(result.current.status.type).toBe('error')
  })
})

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ChatWindowState } from '../../contexts/ChatContext'
import { ControllableFakeWebSocket, chatWrapper as wrapper, checkLoginResponse } from '../../testUtils/chatTestHarness'
import { ChatWindow } from './ChatWindow'
import { ChatWindowsLayer } from './ChatWindowsLayer'

function makeState(overrides: Partial<ChatWindowState> = {}): ChatWindowState {
  return {
    conversationId: 5,
    isGroup: false,
    title: 'bob',
    messages: [],
    offset: 0,
    loadingHistory: false,
    typingUsers: new Set(),
    readStates: {},
    ...overrides,
  }
}

async function setup(myUsername = 'alice') {
  ControllableFakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(checkLoginResponse(myUsername))))

  const socketReady = waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  return { socketReady }
}

// Renders ChatWindow with a hand-crafted state prop — fine for pure
// rendering/interaction assertions, since ChatWindow reads its content
// entirely from that prop and only reaches into ChatContext for the action
// callbacks (sendMessage/sendTyping/etc., which fire their WebSocket sends
// unconditionally regardless of whether the context's own openWindows
// happens to have a matching entry).
async function renderWindow(state: ChatWindowState, myUsername = 'alice') {
  const { socketReady } = await setup(myUsername)
  const view = render(<ChatWindow state={state} />, { wrapper })
  await socketReady
  const socket = ControllableFakeWebSocket.instances[0]
  act(() => socket.simulateOpen())
  return { socket, ...view }
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('ChatWindow', () => {
  it('shows "Chat with <title>" for a direct conversation', async () => {
    await renderWindow(makeState({ isGroup: false, title: 'bob' }))
    expect(screen.getByRole('heading', { name: 'Chat with bob' })).toBeInTheDocument()
  })

  it('shows just the group name for a group conversation', async () => {
    await renderWindow(makeState({ isGroup: true, title: 'Trip Planning' }))
    expect(screen.getByRole('heading', { name: 'Trip Planning' })).toBeInTheDocument()
  })

  it('renders each message with its sender', async () => {
    await renderWindow(makeState({ messages: [{ id: 1, from: 'bob', message: 'hello there', timestamp: '2026-01-01T00:00:00Z' }] }))
    const messageRow = screen.getByText('hello there').closest('p')
    expect(messageRow).not.toBeNull()
    expect(messageRow).toHaveTextContent('bob')
  })

  it('sends the draft on Enter and clears it', async () => {
    const { socket } = await renderWindow(makeState())
    const textarea = screen.getByPlaceholderText('Type your message')
    await userEvent.type(textarea, 'hello there{enter}')

    await waitFor(() => {
      const sendFrame = socket.sent.find((f) => (JSON.parse(f) as { type: string }).type === 'new-message')
      expect(sendFrame).toBeDefined()
      expect((JSON.parse(sendFrame!) as { payload: unknown }).payload).toEqual({ conversation_id: 5, message: 'hello there' })
    })
    expect(textarea).toHaveValue('')
  })

  it('does not send on Shift+Enter, inserts a newline in the draft instead', async () => {
    const { socket } = await renderWindow(makeState())
    const textarea = screen.getByPlaceholderText('Type your message') as HTMLTextAreaElement
    await userEvent.type(textarea, 'line one{Shift>}{enter}{/Shift}line two')

    expect(socket.sent.some((f) => (JSON.parse(f) as { type: string }).type === 'new-message')).toBe(false)
    expect(textarea.value).toContain('line one')
    expect(textarea.value).toContain('line two')
  })

  it('sends the draft when clicking Send', async () => {
    const { socket } = await renderWindow(makeState())
    await userEvent.type(screen.getByPlaceholderText('Type your message'), 'via button')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() => {
      expect(socket.sent.some((f) => (JSON.parse(f) as { type: string }).type === 'new-message')).toBe(true)
    })
  })

  it('does not send an empty or whitespace-only draft', async () => {
    const { socket } = await renderWindow(makeState())
    await userEvent.type(screen.getByPlaceholderText('Type your message'), '   {enter}')
    expect(socket.sent.some((f) => (JSON.parse(f) as { type: string }).type === 'new-message')).toBe(false)
  })

  it('typing sends a typing event with the conversation id', async () => {
    const { socket } = await renderWindow(makeState())
    await userEvent.type(screen.getByPlaceholderText('Type your message'), 'h')

    await waitFor(() => {
      const typingFrame = socket.sent.find((f) => (JSON.parse(f) as { type: string }).type === 'typing')
      expect(typingFrame).toBeDefined()
      expect((JSON.parse(typingFrame!) as { payload: unknown }).payload).toEqual({ conversation_id: 5 })
    })
  })

  it('sends stop-typing after the idle timeout elapses', async () => {
    const { socket } = await renderWindow(makeState())
    vi.useFakeTimers()

    fireEvent.change(screen.getByPlaceholderText('Type your message'), { target: { value: 'h' } })
    expect(socket.sent.some((f) => (JSON.parse(f) as { type: string }).type === 'stop-typing')).toBe(false)

    act(() => vi.advanceTimersByTime(1000))
    expect(socket.sent.some((f) => (JSON.parse(f) as { type: string }).type === 'stop-typing')).toBe(true)
  })

  it('close button is present and clickable', async () => {
    await renderWindow(makeState())
    // closeChat's effect lives in ChatContext state, not observable from
    // ChatWindow's own DOM when rendered with a standalone state prop —
    // this just confirms the control exists and doesn't throw on click.
    const closeButton = screen.getByRole('button', { name: 'x' })
    await userEvent.click(closeButton)
    expect(closeButton).toBeInTheDocument()
  })

  it('shows the typing indicator image when someone else is typing', async () => {
    await renderWindow(makeState({ typingUsers: new Set(['bob']) }))
    expect(document.querySelector('img[id^="typing-indicator-"]')).not.toBeNull()
  })

  it('does not show the typing indicator image when no one is typing', async () => {
    await renderWindow(makeState())
    expect(document.querySelector('img[id^="typing-indicator-"]')).toBeNull()
  })

  it('a group typing indicator lists who is typing', async () => {
    await renderWindow(makeState({ isGroup: true, typingUsers: new Set(['bob', 'carol']) }))
    expect(screen.getByText('bob, carol typing...')).toBeInTheDocument()
  })

  it('a direct-chat typing indicator shows no names, just the icon', async () => {
    await renderWindow(makeState({ isGroup: false, typingUsers: new Set(['bob']) }))
    expect(screen.queryByText(/typing/)).not.toBeInTheDocument()
  })

  it('shows "Seen by" once another member has read up to the latest message', async () => {
    await renderWindow(
      makeState({
        messages: [{ id: 42, from: 'alice', message: 'hi', timestamp: '2026-01-01T00:00:00Z' }],
        readStates: { bob: 42 },
      }),
    )
    expect(screen.getByText('Seen by bob')).toBeInTheDocument()
  })

  it('does not show "Seen by" when no one has caught up to the latest message', async () => {
    await renderWindow(
      makeState({
        messages: [{ id: 42, from: 'alice', message: 'hi', timestamp: '2026-01-01T00:00:00Z' }],
        readStates: { bob: 10 },
      }),
    )
    expect(screen.queryByText(/Seen by/)).not.toBeInTheDocument()
  })

  it('does not show "Seen by" for an unconfirmed (locally-echoed) latest message', async () => {
    await renderWindow(
      makeState({
        messages: [{ id: 0, from: 'alice', message: 'just sent', timestamp: Date.now() }],
        readStates: { bob: 999 },
      }),
    )
    expect(screen.queryByText(/Seen by/)).not.toBeInTheDocument()
  })

  it('scrolling to the top of an open conversation requests more history', async () => {
    const { socketReady } = await setup()
    render(<ChatWindowsLayer />, { wrapper })
    await socketReady
    const socket = ControllableFakeWebSocket.instances[0]
    act(() => socket.simulateOpen())

    act(() =>
      socket.simulateMessage('chat-opened', {
        conversation_id: 5,
        is_group: false,
        members: [
          { user_id: 1, username: 'alice' },
          { user_id: 2, username: 'bob' },
        ],
        read_states: [],
      }),
    )
    await waitFor(() => expect(document.getElementById('chat:5')).not.toBeNull())

    // chat-opened triggers an initial get-chat-history fetch for page 1 —
    // that has to resolve (clearing loadingHistory) before a scroll-
    // triggered loadMoreHistory call will do anything, matching the real
    // guard in ChatContext.
    act(() => socket.simulateMessage('chat_history', []))

    const messagesEl = document.getElementById('chat-messages-5')!
    Object.defineProperty(messagesEl, 'scrollTop', { value: 0, writable: true })
    act(() => {
      messagesEl.dispatchEvent(new Event('scroll', { bubbles: true }))
    })

    await waitFor(() => {
      const historyFrame = socket.sent.find((f) => (JSON.parse(f) as { type: string }).type === 'get-more-chat-history')
      expect(historyFrame).toBeDefined()
      expect((JSON.parse(historyFrame!) as { payload: unknown }).payload).toEqual({ conversation_id: 5, offset: 10, limit: 10 })
    })
  })
})

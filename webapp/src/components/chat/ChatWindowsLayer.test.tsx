import { act, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ControllableFakeWebSocket, chatWrapper as wrapper, checkLoginResponse } from '../../testUtils/chatTestHarness'
import { ChatWindowsLayer } from './ChatWindowsLayer'

async function setup(myUsername = 'alice') {
  ControllableFakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(checkLoginResponse(myUsername))))

  const view = render(<ChatWindowsLayer />, { wrapper })
  await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  const socket = ControllableFakeWebSocket.instances[0]
  act(() => socket.simulateOpen())
  return { socket, ...view }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ChatWindowsLayer', () => {
  it('renders nothing when no conversation windows are open', async () => {
    await setup()
    expect(document.querySelectorAll('.chat-window')).toHaveLength(0)
  })

  it('renders one chat window per open conversation, pushed by chat-opened', async () => {
    const { socket } = await setup()

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
    act(() =>
      socket.simulateMessage('chat-opened', {
        conversation_id: 9,
        is_group: true,
        name: 'Trip Planning',
        members: [
          { user_id: 1, username: 'alice' },
          { user_id: 3, username: 'carol' },
        ],
        read_states: [],
      }),
    )

    await waitFor(() => expect(document.querySelectorAll('.chat-window')).toHaveLength(2))
    expect(document.getElementById('chat:5')).not.toBeNull()
    expect(document.getElementById('chat:9')).not.toBeNull()
    expect(screen.getByText('Trip Planning')).toBeInTheDocument()
  })
})

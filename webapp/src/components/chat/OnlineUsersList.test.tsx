import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ControllableFakeWebSocket, chatWrapper as wrapper, checkLoginResponse } from '../../testUtils/chatTestHarness'
import { OnlineUsersList } from './OnlineUsersList'

async function setup(myUsername = 'alice') {
  ControllableFakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(checkLoginResponse(myUsername))))

  const view = render(<OnlineUsersList />, { wrapper })
  await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  const socket = ControllableFakeWebSocket.instances[0]
  act(() => socket.simulateOpen())
  return { socket, ...view }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('OnlineUsersList', () => {
  it('renders every online user', async () => {
    const { socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true, bob: true, carol: true }))

    expect(await screen.findByText('bob')).toBeInTheDocument()
    expect(screen.getByText('carol')).toBeInTheDocument()
    expect(screen.getByText('alice')).toBeInTheDocument()
  })

  it('clicking another user sends open-direct-chat for them', async () => {
    const { socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true, bob: true }))

    await userEvent.click(await screen.findByText('bob'))

    await waitFor(() => {
      const openRequest = socket.sent.find((frame) => (JSON.parse(frame) as { type: string }).type === 'open-direct-chat')
      expect(openRequest).toBeDefined()
      expect((JSON.parse(openRequest!) as { payload: unknown }).payload).toEqual({ username: 'bob' })
    })
  })

  it('clicking your own name does not open a chat with yourself', async () => {
    const { socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true }))

    await userEvent.click(await screen.findByText('alice'))

    expect(socket.sent.some((frame) => (JSON.parse(frame) as { type: string }).type === 'open-direct-chat')).toBe(false)
  })

  it('shows an unread badge next to a user with an unread message', async () => {
    const { socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true, bob: true }))
    act(() =>
      socket.simulateMessage('sent-message', { id: 1, conversation_id: 5, from: 'bob', message: 'hi', sent: '2026-01-01T00:00:00Z' }),
    )

    const bobItem = (await screen.findByText('bob')).closest('li')
    expect(bobItem).not.toBeNull()
    await waitFor(() => expect(bobItem!.querySelector('.msg-alert')).not.toBeNull())
  })

  it('does not show an unread badge for a user with no unread messages', async () => {
    const { socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true, bob: true }))

    const bobItem = (await screen.findByText('bob')).closest('li')
    expect(bobItem!.querySelector('.msg-alert')).toBeNull()
  })
})

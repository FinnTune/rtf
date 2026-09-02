import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ControllableFakeWebSocket, chatWrapper as wrapper, checkLoginResponse } from '../../testUtils/chatTestHarness'
import { GroupChatsPanel } from './GroupChatsPanel'

async function setup(myUsername = 'alice') {
  ControllableFakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(checkLoginResponse(myUsername))))

  const view = render(<GroupChatsPanel />, { wrapper })
  await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  const socket = ControllableFakeWebSocket.instances[0]
  act(() => socket.simulateOpen())
  return { socket, ...view }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('GroupChatsPanel', () => {
  it('toggles the new-group form open and closed', async () => {
    await setup()
    expect(screen.queryByPlaceholderText('Group name')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: '+ New Group' }))
    expect(screen.getByPlaceholderText('Group name')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByPlaceholderText('Group name')).not.toBeInTheDocument()
  })

  it('lists online users as selectable members in the new-group form', async () => {
    const { socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true, bob: true, carol: true }))

    await userEvent.click(screen.getByRole('button', { name: '+ New Group' }))

    expect(screen.getByRole('checkbox', { name: 'bob' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'carol' })).toBeInTheDocument()
  })

  it('disables Create Group until a name and at least one member are chosen', async () => {
    const { socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true, bob: true }))
    await userEvent.click(screen.getByRole('button', { name: '+ New Group' }))

    const createButton = screen.getByRole('button', { name: 'Create Group' })
    expect(createButton).toBeDisabled()

    await userEvent.type(screen.getByPlaceholderText('Group name'), 'Trip Planning')
    expect(createButton).toBeDisabled()

    await userEvent.click(screen.getByRole('checkbox', { name: 'bob' }))
    expect(createButton).toBeEnabled()
  })

  it('creating a group sends create-group-chat with the name and selected members, then resets the form', async () => {
    const { socket } = await setup()
    act(() => socket.simulateMessage('users-online', { alice: true, bob: true, carol: true }))
    await userEvent.click(screen.getByRole('button', { name: '+ New Group' }))

    await userEvent.type(screen.getByPlaceholderText('Group name'), 'Trip Planning')
    await userEvent.click(screen.getByRole('checkbox', { name: 'bob' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'carol' }))
    await userEvent.click(screen.getByRole('button', { name: 'Create Group' }))

    const createRequest = socket.sent.find((frame) => (JSON.parse(frame) as { type: string }).type === 'create-group-chat')
    expect(createRequest).toBeDefined()
    const payload = (JSON.parse(createRequest!) as { payload: { name: string; usernames: string[] } }).payload
    expect(payload.name).toBe('Trip Planning')
    expect(payload.usernames.sort()).toEqual(['bob', 'carol'])

    // The form closes and resets after creating.
    expect(screen.queryByPlaceholderText('Group name')).not.toBeInTheDocument()
  })

  it('lists known group chats and shows an unread badge', async () => {
    const { socket } = await setup()

    act(() =>
      socket.simulateMessage('conversations-list', [
        {
          conversation_id: 9,
          is_group: true,
          name: 'Trip Planning',
          members: [
            { user_id: 1, username: 'alice' },
            { user_id: 2, username: 'bob' },
          ],
          read_states: [],
        },
      ]),
    )
    expect(await screen.findByText('Trip Planning')).toBeInTheDocument()
    expect(screen.getByText('Trip Planning').closest('li')!.querySelector('.msg-alert')).toBeNull()

    act(() =>
      socket.simulateMessage('sent-message', { id: 1, conversation_id: 9, from: 'bob', message: 'hi group', sent: '2026-01-01T00:00:00Z' }),
    )
    await waitFor(() => expect(screen.getByText('Trip Planning').closest('li')!.querySelector('.msg-alert')).not.toBeNull())
  })

  it('clicking a group chat requests its history', async () => {
    const { socket } = await setup()
    act(() =>
      socket.simulateMessage('conversations-list', [
        {
          conversation_id: 9,
          is_group: true,
          name: 'Trip Planning',
          members: [
            { user_id: 1, username: 'alice' },
            { user_id: 2, username: 'bob' },
          ],
          read_states: [],
        },
      ]),
    )
    await userEvent.click(await screen.findByText('Trip Planning'))

    await waitFor(() => {
      const historyRequest = socket.sent.find((frame) => (JSON.parse(frame) as { type: string }).type === 'get-chat-history')
      expect(historyRequest).toBeDefined()
      expect((JSON.parse(historyRequest!) as { payload: unknown }).payload).toEqual({ conversation_id: 9, offset: 0, limit: 10 })
    })
  })
})

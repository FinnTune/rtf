import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { NotificationsProvider } from '../../contexts/NotificationsContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { WebSocketProvider } from '../../contexts/WebSocketContext'
import { ControllableFakeWebSocket, checkLoginResponse, requestUrl } from '../../testUtils/chatTestHarness'
import { NotificationsBell } from './NotificationsBell'

function notificationsListResponse(notifications: object[], unreadCount: number) {
  return new Response(JSON.stringify({ notifications, unread_count: unreadCount }), {
    status: 200,
    headers: { 'X-Total-Count': String(notifications.length) },
  })
}

function makeNotification(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 1,
    post_id: 7,
    post_title: 'My Post',
    comment_id: 50,
    actor_username: 'alice',
    created_at: '2026-01-01 00:00:00',
    read: false,
    ...overrides,
  }
}

async function renderBell(notifications: object[], unreadCount: number) {
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
      if (url.startsWith('/notifications')) return notificationsListResponse(notifications, unreadCount)
      if (url.startsWith('/markNotificationsRead')) return new Response(null, { status: 200 })
      throw new Error('Unexpected fetch: ' + url)
    }),
  )

  const result = render(
    <MemoryRouter initialEntries={['/']}>
      <StatusMessageProvider>
        <AuthProvider>
          <WebSocketProvider>
            <NotificationsProvider>
              {/* NotificationsBell itself sits outside the swapped route,
                  matching its real usage in Topbar.tsx (rendered once,
                  persisting across route changes) — this is what lets
                  "the dropdown closed" be verified independently of "the
                  page navigated", rather than the dropdown's content merely
                  disappearing because the whole bell unmounted. */}
              <NotificationsBell />
              <Routes>
                <Route path="/" element={<p>Feed page</p>} />
                <Route path="/posts/:id" element={<p>Post detail page</p>} />
              </Routes>
            </NotificationsProvider>
          </WebSocketProvider>
        </AuthProvider>
      </StatusMessageProvider>
    </MemoryRouter>,
  )

  // Settles the WS connection before returning, the same way every other
  // WS-dependent test in this app does (see e.g. GroupChatsPanel.test.tsx)
  // — otherwise the connection sits "connecting" indefinitely across the
  // whole test, leaving its in-flight checkLogin promise still unresolved
  // (and its eventual .then() still pending) by the time a later test's
  // afterEach has already restored the real global fetch.
  await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  act(() => ControllableFakeWebSocket.instances[0].simulateOpen())

  return result
}

afterEach(() => {
  vi.unstubAllGlobals()
  ControllableFakeWebSocket.instances = []
})

describe('NotificationsBell', () => {
  it('shows no unread badge when there are no unread notifications', async () => {
    await renderBell([makeNotification({ read: true })], 0)
    await screen.findByRole('button', { name: 'Notifications' })
    expect(screen.queryByLabelText(/unread notifications/)).not.toBeInTheDocument()
  })

  it('shows the unread count badge and the list collapsed by default', async () => {
    await renderBell([makeNotification()], 1)
    expect(await screen.findByLabelText('1 unread notifications')).toHaveTextContent('1')
    expect(screen.queryByText(/commented on/)).not.toBeInTheDocument()
  })

  it('opens to show the notification list on toggle click', async () => {
    await renderBell([makeNotification({ actor_username: 'bob', post_title: 'Great Thread' })], 1)
    await waitFor(() => expect(screen.getByRole('button', { name: /Notifications/ })).toBeInTheDocument())

    await userEvent.click(screen.getByRole('button', { name: /Notifications/ }))
    expect(screen.getByText(/bob/)).toBeInTheDocument()
    expect(screen.getByText(/Great Thread/)).toBeInTheDocument()
  })

  it('clicking a notification marks it read, closes the dropdown, and navigates to the post', async () => {
    await renderBell([makeNotification({ id: 9, post_id: 42 })], 1)
    await userEvent.click(await screen.findByRole('button', { name: /Notifications/ }))

    await userEvent.click(screen.getByText(/commented on/))

    expect(await screen.findByText('Post detail page')).toBeInTheDocument()
    // The dropdown (and its content) closed along with the navigation.
    expect(screen.queryByText(/commented on/)).not.toBeInTheDocument()
  })

  it('"Mark all read" clears the unread badge without navigating', async () => {
    await renderBell([makeNotification({ read: false }), makeNotification({ id: 2, read: false })], 2)
    await userEvent.click(await screen.findByRole('button', { name: /Notifications/ }))

    await userEvent.click(screen.getByRole('button', { name: 'Mark all read' }))

    await waitFor(() => expect(screen.queryByLabelText(/unread notifications/)).not.toBeInTheDocument())
    expect(screen.queryByText('Post detail page')).not.toBeInTheDocument()
  })

  it('shows an empty state when there are no notifications at all', async () => {
    await renderBell([], 0)
    await act(async () => {
      await userEvent.click(await screen.findByRole('button', { name: 'Notifications' }))
    })
    expect(screen.getByText('No notifications yet.')).toBeInTheDocument()
  })
})

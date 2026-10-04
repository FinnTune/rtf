import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ControllableFakeWebSocket, checkLoginResponse, requestUrl } from '../testUtils/chatTestHarness'
import { AuthProvider } from './AuthContext'
import { NotificationsProvider, useNotifications } from './NotificationsContext'
import { StatusMessageProvider } from './StatusMessageContext'
import { WebSocketProvider } from './WebSocketContext'

function notificationsWrapper({ children }: { children: ReactNode }) {
  return (
    <StatusMessageProvider>
      <AuthProvider>
        <WebSocketProvider>
          <NotificationsProvider>{children}</NotificationsProvider>
        </WebSocketProvider>
      </AuthProvider>
    </StatusMessageProvider>
  )
}

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

afterEach(() => {
  vi.unstubAllGlobals()
  ControllableFakeWebSocket.instances = []
})

describe('NotificationsContext', () => {
  it('loads the initial notification list and unread count on mount', async () => {
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/notifications')) {
          return notificationsListResponse([makeNotification({ id: 1 }), makeNotification({ id: 2, read: true })], 1)
        }
        throw new Error('Unexpected fetch: ' + url)
      }),
    )

    const { result } = renderHook(() => useNotifications(), { wrapper: notificationsWrapper })

    await waitFor(() => expect(result.current.notifications).toHaveLength(2))
    expect(result.current.unreadCount).toBe(1)
  })

  it('a notification-added push prepends the new item and increments unreadCount', async () => {
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = requestUrl(input)
        if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
        if (url.startsWith('/notifications')) return notificationsListResponse([makeNotification({ id: 1 })], 1)
        throw new Error('Unexpected fetch: ' + url)
      }),
    )

    const { result } = renderHook(() => useNotifications(), { wrapper: notificationsWrapper })
    await waitFor(() => expect(result.current.notifications).toHaveLength(1))

    await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
    const socket = ControllableFakeWebSocket.instances[0]
    act(() => socket.simulateOpen())
    act(() => socket.simulateMessage('notification-added', makeNotification({ id: 2, post_title: 'Another Post' })))

    expect(result.current.notifications).toHaveLength(2)
    expect(result.current.notifications[0].id).toBe(2)
    expect(result.current.unreadCount).toBe(2)
  })

  it('markRead marks just that notification read, decrements unreadCount, and calls the API with its id', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
      if (url.startsWith('/notifications')) {
        return notificationsListResponse([makeNotification({ id: 1 }), makeNotification({ id: 2 })], 2)
      }
      if (url.startsWith('/markNotificationsRead')) return new Response(null, { status: 200 })
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    vi.stubGlobal('fetch', fetchMock)

    const { result } = renderHook(() => useNotifications(), { wrapper: notificationsWrapper })
    await waitFor(() => expect(result.current.notifications).toHaveLength(2))

    act(() => result.current.markRead(1))

    expect(result.current.notifications.find((n) => n.id === 1)?.read).toBe(true)
    expect(result.current.notifications.find((n) => n.id === 2)?.read).toBe(false)
    expect(result.current.unreadCount).toBe(1)

    await waitFor(() => {
      const call = vi.mocked(fetch).mock.calls.find(([input]) => requestUrl(input as string | URL | Request).startsWith('/markNotificationsRead'))
      expect(call).toBeDefined()
      expect(JSON.parse((call![1] as RequestInit).body as string)).toEqual({ id: 1 })
    })
  })

  it('markRead on an already-read notification is a no-op (no API call, unreadCount unchanged)', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
      if (url.startsWith('/notifications')) return notificationsListResponse([makeNotification({ id: 1, read: true })], 0)
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    vi.stubGlobal('fetch', fetchMock)

    const { result } = renderHook(() => useNotifications(), { wrapper: notificationsWrapper })
    await waitFor(() => expect(result.current.notifications).toHaveLength(1))

    act(() => result.current.markRead(1))

    expect(result.current.unreadCount).toBe(0)
    expect(fetchMock.mock.calls.some(([input]) => requestUrl(input as string | URL | Request).startsWith('/markNotificationsRead'))).toBe(false)
  })

  it('markAllRead marks every notification read, zeroes unreadCount, and calls the API with no id', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('alice')
      if (url.startsWith('/notifications')) {
        return notificationsListResponse([makeNotification({ id: 1 }), makeNotification({ id: 2 })], 2)
      }
      if (url.startsWith('/markNotificationsRead')) return new Response(null, { status: 200 })
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    vi.stubGlobal('fetch', fetchMock)

    const { result } = renderHook(() => useNotifications(), { wrapper: notificationsWrapper })
    await waitFor(() => expect(result.current.notifications).toHaveLength(2))

    act(() => result.current.markAllRead())

    expect(result.current.notifications.every((n) => n.read)).toBe(true)
    expect(result.current.unreadCount).toBe(0)

    await waitFor(() => {
      const call = vi.mocked(fetch).mock.calls.find(([input]) => requestUrl(input as string | URL | Request).startsWith('/markNotificationsRead'))
      expect(call).toBeDefined()
      expect((call![1] as RequestInit).body).toBeUndefined()
    })
  })
})

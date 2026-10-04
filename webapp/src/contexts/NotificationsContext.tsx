import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { getNotifications, markNotificationsRead } from '../api/notifications'
import { useAuth } from './AuthContext'
import { useStatusMessage } from './StatusMessageContext'
import { useWebSocket } from './WebSocketContext'
import type { NotificationItem } from '../types'

const INITIAL_PAGE_SIZE = 20

interface NotificationsContextValue {
  notifications: NotificationItem[]
  unreadCount: number
  markRead: (id: number) => void
  markAllRead: () => void
}

const NotificationsContext = createContext<NotificationsContextValue | null>(null)

export function NotificationsProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const { subscribe } = useWebSocket()
  const { showMessage } = useStatusMessage()

  const [notifications, setNotifications] = useState<NotificationItem[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  // markRead needs to know whether the item it's marking was already read
  // (to avoid double-decrementing unreadCount) without taking a
  // `notifications` dependency itself, which would change this callback's
  // identity on every list update (a live push, another markRead, ...) —
  // same ref-mirrors-state shape as ChatContext's openWindowsRef.
  const notificationsRef = useRef(notifications)
  useEffect(() => {
    notificationsRef.current = notifications
  }, [notifications])

  const userID = user?.id

  // Loads this user's own recent notifications once per login — depends on
  // userID specifically (a primitive), not the whole user object, which
  // AuthContext hands out a new reference for on every checkLogin refresh
  // (e.g. the WebSocket reconnect flow's own periodic poll) even when it's
  // still the same logged-in user — that would otherwise refetch far more
  // often than "once per login".
  useEffect(() => {
    if (!userID) {
      setNotifications([])
      setUnreadCount(0)
      return
    }
    getNotifications(0, INITIAL_PAGE_SIZE)
      .then(({ notifications, unreadCount }) => {
        setNotifications(notifications)
        setUnreadCount(unreadCount)
      })
      .catch((error: unknown) => {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
  }, [userID, showMessage])

  // Live push for a notification landing while this tab is open — see
  // broadcastNotificationAdded's own doc comment for why this is targeted
  // (every other connected client for this one user, not a feed-wide
  // broadcast like comment-added).
  useEffect(() => {
    return subscribe('notification-added', (payload) => {
      const notification = payload as NotificationItem
      setNotifications((prev) => [notification, ...prev])
      setUnreadCount((prev) => prev + 1)
    })
  }, [subscribe])

  const markRead = useCallback((id: number) => {
    const target = notificationsRef.current.find((n) => n.id === id)
    if (!target || target.read) return
    setNotifications((prev) => prev.map((n) => (n.id === id ? { ...n, read: true } : n)))
    setUnreadCount((count) => Math.max(0, count - 1))
    void markNotificationsRead(id)
    // Not awaited/rolled back on failure — a missed mark-read is a minor,
    // self-correcting annoyance (it just stays visually unread until the
    // next successful one), not worth the extra state-reconciliation
    // complexity of an optimistic-update rollback like useReaction's.
  }, [])

  const markAllRead = useCallback(() => {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })))
    setUnreadCount(0)
    void markNotificationsRead()
  }, [])

  return (
    <NotificationsContext.Provider value={{ notifications, unreadCount, markRead, markAllRead }}>
      {children}
    </NotificationsContext.Provider>
  )
}

export function useNotifications(): NotificationsContextValue {
  const ctx = useContext(NotificationsContext)
  if (!ctx) {
    throw new Error('useNotifications must be used within a NotificationsProvider')
  }
  return ctx
}

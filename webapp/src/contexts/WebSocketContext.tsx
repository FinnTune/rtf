import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { checkLogin } from '../api/auth'
import { useAuth } from './AuthContext'
import { useStatusMessage } from './StatusMessageContext'

export type WSStatus = 'idle' | 'connecting' | 'open' | 'reconnecting' | 'closed'

type Handler = (payload: unknown) => void

interface WebSocketContextValue {
  status: WSStatus
  send: (type: string, payload: unknown) => void
  subscribe: (type: string, handler: Handler) => () => void
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null)

const MAX_RECONNECT_ATTEMPTS = 3

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const { user, refresh } = useAuth()
  const { showMessage } = useStatusMessage()
  const [status, setStatus] = useState<WSStatus>('idle')

  const socketRef = useRef<WebSocket | null>(null)
  const listenersRef = useRef(new Map<string, Set<Handler>>())
  // Set right before a deliberate close (logout, or replacing a connection
  // during reconnect) so the resulting close event isn't mistaken for a
  // dropped connection and doesn't trigger a reconnect attempt.
  const expectedCloseRef = useRef(false)
  const reconnectAttemptsRef = useRef(0)
  // openConnection's onclose handler needs to call attemptReconnect, but
  // attemptReconnect (below) also calls openConnection — a real mutual
  // reference, not one that's safe to statically verify. Route the call
  // through a ref (kept current by the effect further down) instead of a
  // direct reference.
  const attemptReconnectRef = useRef<() => void>(() => {})

  const dispatch = useCallback((type: string, payload: unknown) => {
    listenersRef.current.get(type)?.forEach((handler) => handler(payload))
  }, [])

  const send = useCallback((type: string, payload: unknown) => {
    const socket = socketRef.current
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type, payload }))
    }
  }, [])

  const subscribe = useCallback((type: string, handler: Handler) => {
    const listeners = listenersRef.current
    if (!listeners.has(type)) {
      listeners.set(type, new Set())
    }
    listeners.get(type)!.add(handler)
    return () => {
      listeners.get(type)?.delete(handler)
    }
  }, [])

  // Opens a connection with a given (freshly-minted, single-use) OTP and
  // wires up its handlers. Shared by the initial connect and every
  // reconnect attempt — each of those mints its own fresh OTP first, since
  // reusing one across attempts would never work anyway (OTPs are
  // consumed on the handshake that uses them, and expire in ~5s besides).
  const openConnection = useCallback(
    (otp: string) => {
      // The production server always runs TLS (main.go's ListenAndServeTLS),
      // so this is wss:// there — but the Vite dev server runs plain HTTP
      // on :5173 with no TLS at all, where it must be ws:// instead.
      // Deriving it from the page's own protocol handles both.
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const socket = new WebSocket(`${wsProtocol}//${window.location.host}/ws?otp=${otp}`)
      socketRef.current = socket

      socket.onopen = () => {
        reconnectAttemptsRef.current = 0
        setStatus('open')
        // The server ignores every field in this payload — the connection
        // is already bound to the OTP-verified identity — so there's
        // nothing meaningful to send beyond the event type itself.
        socket.send(JSON.stringify({ type: 'user-connect', payload: {} }))
        // Existing group chats (unlike a direct chat) can't be rediscovered
        // just by clicking an online user, so fetch them once up front.
        socket.send(JSON.stringify({ type: 'get-conversations', payload: {} }))
      }

      socket.onclose = () => {
        if (expectedCloseRef.current) {
          expectedCloseRef.current = false
          setStatus('closed')
          return
        }
        showMessage('Connection lost. Reconnecting...', 'error')
        setStatus('reconnecting')
        attemptReconnectRef.current()
      }

      socket.onmessage = (event: MessageEvent<string>) => {
        try {
          const parsed = JSON.parse(event.data) as { type: string; payload: unknown }
          dispatch(parsed.type, parsed.payload)
        } catch {
          // Malformed frame — nothing sensible to recover to, just drop it.
        }
      }
    },
    [dispatch, showMessage],
  )

  const attemptReconnect = useCallback(() => {
    if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
      showMessage('Connection lost. Please refresh the page.', 'error')
      return
    }
    reconnectAttemptsRef.current += 1
    const attempt = reconnectAttemptsRef.current
    setTimeout(() => {
      void refresh().then((freshUser) => {
        if (freshUser) {
          showMessage('Reconnected.', 'success')
          openConnection(freshUser.otp)
        } else {
          showMessage('Your session has ended. Please log in again.', 'error')
        }
      })
    }, 1000 * attempt)
  }, [refresh, showMessage, openConnection])

  useEffect(() => {
    attemptReconnectRef.current = attemptReconnect
  }, [attemptReconnect])

  useEffect(() => {
    if (!user) {
      setStatus('idle')
      return
    }

    let cancelled = false
    setStatus('connecting')

    // Always mint a fresh OTP right before connecting, rather than reusing
    // user.otp from React state — that value may already be stale/consumed
    // by the time this effect runs (e.g. React re-invoking this effect
    // without a real state change), and OTPs don't tolerate reuse anyway.
    void checkLogin().then((fresh) => {
      if (cancelled || !fresh) return
      openConnection(fresh.otp)
    })

    return () => {
      cancelled = true
      if (socketRef.current) {
        expectedCloseRef.current = true
        socketRef.current.close()
        socketRef.current = null
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- keyed on identity (username), not the ever-changing otp/object reference
  }, [user?.username])

  return <WebSocketContext.Provider value={{ status, send, subscribe }}>{children}</WebSocketContext.Provider>
}

export function useWebSocket(): WebSocketContextValue {
  const ctx = useContext(WebSocketContext)
  if (!ctx) {
    throw new Error('useWebSocket must be used within a WebSocketProvider')
  }
  return ctx
}

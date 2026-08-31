import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useAuth } from './AuthContext'
import { useWebSocket } from './WebSocketContext'
import type { ChatMessageVM } from '../types'

const HISTORY_PAGE_SIZE = 10

export interface ChatWindowState {
  messages: ChatMessageVM[]
  offset: number
  loadingHistory: boolean
  isOtherTyping: boolean
}

interface ChatContextValue {
  // Sorted unread-first (in the order each became unread), then the rest
  // alphabetically — the original app intended this via a module-level
  // `alertUsers` array, but nothing in it ever actually populated that
  // array, so the real app never did this in practice. Fixed here since
  // real, persistent React state naturally does the right thing instead.
  onlineUsers: string[]
  unreadUsers: Set<string>
  openWindows: Record<string, ChatWindowState>
  openChat: (username: string) => void
  closeChat: (username: string) => void
  sendMessage: (username: string, text: string) => void
  loadMoreHistory: (username: string) => void
  sendTyping: (username: string) => void
  sendStopTyping: (username: string) => void
}

const ChatContext = createContext<ChatContextValue | null>(null)

interface RawChatHistoryItem {
  from: string
  to: string
  message: string
  created_at: string
}

interface RawSentMessage {
  from: string
  to: string
  message: string
  sent: string
}

interface RawTypingEvent {
  from: string
  to: string
}

export function ChatProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const { subscribe, send } = useWebSocket()
  const myUsername = user?.username ?? ''

  const [onlineUsernames, setOnlineUsernames] = useState<string[]>([])
  const [unreadUsers, setUnreadUsers] = useState<Set<string>>(new Set())
  const [openWindows, setOpenWindows] = useState<Record<string, ChatWindowState>>({})

  // Subscription handlers close over this to know which conversations are
  // currently open, without needing to resubscribe every time a window
  // opens/closes.
  const openWindowsRef = useRef(openWindows)
  useEffect(() => {
    openWindowsRef.current = openWindows
  }, [openWindows])

  const markUnread = useCallback(
    (username: string) => {
      if (username === myUsername) return
      setUnreadUsers((prev) => (prev.has(username) ? prev : new Set(prev).add(username)))
    },
    [myUsername],
  )

  useEffect(() => {
    const unsubUsers = subscribe('users-online', (payload) => {
      setOnlineUsernames(Object.keys(payload as Record<string, boolean>))
    })

    const unsubSent = subscribe('sent-message', (payload) => {
      const data = payload as RawSentMessage
      const otherParty = data.from === myUsername ? data.to : data.from
      const vm: ChatMessageVM = { from: data.from, to: data.to, message: data.message, timestamp: data.sent }
      if (openWindowsRef.current[otherParty]) {
        setOpenWindows((prev) => ({
          ...prev,
          [otherParty]: { ...prev[otherParty], messages: [...prev[otherParty].messages, vm] },
        }))
      } else {
        markUnread(data.from)
      }
    })

    const unsubHistory = subscribe('chat_history', (payload) => {
      const batch = payload as RawChatHistoryItem[] | null
      if (!batch || batch.length === 0) return
      // Every message in one response is for the same conversation (the
      // server scopes the query to the requester's own identity plus one
      // other user) — the first message's "other party" identifies it.
      const first = batch[0]
      const otherParty = first.from === myUsername ? first.to : first.from
      if (!openWindowsRef.current[otherParty]) {
        markUnread(otherParty)
        return
      }
      const vms: ChatMessageVM[] = batch.map((m) => ({ from: m.from, to: m.to, message: m.message, timestamp: m.created_at }))
      setOpenWindows((prev) => ({
        ...prev,
        [otherParty]: { ...prev[otherParty], messages: [...vms, ...prev[otherParty].messages], loadingHistory: false },
      }))
    })

    const unsubTyping = subscribe('typing', (payload) => {
      const data = payload as RawTypingEvent
      if (openWindowsRef.current[data.from]) {
        setOpenWindows((prev) => ({ ...prev, [data.from]: { ...prev[data.from], isOtherTyping: true } }))
      }
    })

    const unsubStopTyping = subscribe('stop-typing', (payload) => {
      const data = payload as RawTypingEvent
      if (openWindowsRef.current[data.from]) {
        setOpenWindows((prev) => ({ ...prev, [data.from]: { ...prev[data.from], isOtherTyping: false } }))
      }
    })

    return () => {
      unsubUsers()
      unsubSent()
      unsubHistory()
      unsubTyping()
      unsubStopTyping()
    }
  }, [subscribe, myUsername, markUnread])

  const openChat = useCallback(
    (username: string) => {
      setUnreadUsers((prev) => {
        if (!prev.has(username)) return prev
        const next = new Set(prev)
        next.delete(username)
        return next
      })
      // Checked against the ref, not a flag mutated inside the setOpenWindows
      // updater below — that updater isn't guaranteed to run synchronously
      // before this line, so a flag set inside it can't be trusted here.
      if (!openWindowsRef.current[username]) {
        setOpenWindows((prev) => ({
          ...prev,
          [username]: { messages: [], offset: 0, loadingHistory: true, isOtherTyping: false },
        }))
        send('get-chat-history', { from: myUsername, to: username, offset: 0, limit: HISTORY_PAGE_SIZE })
      }
    },
    [send, myUsername],
  )

  const closeChat = useCallback((username: string) => {
    setOpenWindows((prev) => {
      if (!(username in prev)) return prev
      const next = { ...prev }
      delete next[username]
      return next
    })
  }, [])

  const sendMessage = useCallback(
    (username: string, text: string) => {
      // The server never echoes a sent message back to its own sender (see
      // ws-manager.go's sendMessage — it only delivers to the recipient) —
      // append it locally, optimistically, or the sender would never see
      // their own message in the window at all.
      const vm: ChatMessageVM = { from: myUsername, to: username, message: text, timestamp: Date.now() }
      setOpenWindows((prev) => {
        const existing = prev[username] ?? { messages: [], offset: 0, loadingHistory: false, isOtherTyping: false }
        return { ...prev, [username]: { ...existing, messages: [...existing.messages, vm] } }
      })
      send('new-message', { message: text, from: myUsername, to: username, sent: Date.now() })
    },
    [send, myUsername],
  )

  const loadMoreHistory = useCallback(
    (username: string) => {
      setOpenWindows((prev) => {
        const existing = prev[username]
        if (!existing || existing.loadingHistory) return prev
        const nextOffset = existing.offset + HISTORY_PAGE_SIZE
        send('get-more-chat-history', { from: myUsername, to: username, offset: nextOffset, limit: HISTORY_PAGE_SIZE })
        return { ...prev, [username]: { ...existing, offset: nextOffset, loadingHistory: true } }
      })
    },
    [send, myUsername],
  )

  const sendTyping = useCallback((username: string) => send('typing', { from: myUsername, to: username }), [send, myUsername])
  const sendStopTyping = useCallback(
    (username: string) => send('stop-typing', { from: myUsername, to: username }),
    [send, myUsername],
  )

  const onlineUsers = useMemo(() => {
    const unread = [...unreadUsers].filter((u) => onlineUsernames.includes(u))
    const rest = onlineUsernames.filter((u) => !unreadUsers.has(u)).sort()
    return [...unread, ...rest]
  }, [onlineUsernames, unreadUsers])

  return (
    <ChatContext.Provider
      value={{
        onlineUsers,
        unreadUsers,
        openWindows,
        openChat,
        closeChat,
        sendMessage,
        loadMoreHistory,
        sendTyping,
        sendStopTyping,
      }}
    >
      {children}
    </ChatContext.Provider>
  )
}

export function useChat(): ChatContextValue {
  const ctx = useContext(ChatContext)
  if (!ctx) {
    throw new Error('useChat must be used within a ChatProvider')
  }
  return ctx
}

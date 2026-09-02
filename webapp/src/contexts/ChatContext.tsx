import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { isTabInBackground, requestNotificationPermission, showBrowserNotification } from '../notifications'
import { useAuth } from './AuthContext'
import { useStatusMessage } from './StatusMessageContext'
import { useWebSocket } from './WebSocketContext'
import type { ChatMessageVM, ConversationInfo } from '../types'

const HISTORY_PAGE_SIZE = 10

export interface ChatWindowState {
  conversationId: number
  isGroup: boolean
  // The group's name, or the other member's username for a direct chat —
  // whatever a window's title bar should show.
  title: string
  messages: ChatMessageVM[]
  offset: number
  loadingHistory: boolean
  // A set, not a single flag, since a group conversation can have more than
  // one person typing at once.
  typingUsers: Set<string>
  // Every OTHER member's read watermark, keyed by username — the current
  // user's own entry isn't tracked here since there's no "seen by me" UI.
  readStates: Record<string, number>
}

interface ChatContextValue {
  // Sorted unread-first (in the order each became unread), then the rest
  // alphabetically.
  onlineUsers: string[]
  unreadConversations: Set<number>
  // Derived from unreadConversations for direct chats specifically — what
  // OnlineUsersList checks to show the "!" badge next to a username.
  unreadUsernames: Set<string>
  openWindows: Record<number, ChatWindowState>
  groupChats: ConversationInfo[]
  openDirectChat: (username: string) => void
  // Opens (or reveals, if already open) a window for a conversation whose
  // full info is already known — e.g. clicking an entry in the group chats
  // list, where unlike a direct chat there's no username to resolve first.
  openConversation: (info: ConversationInfo) => void
  // Same, but by id — for contexts (like a message search result) that
  // only have a conversation_id and rely on it already being in the
  // locally-known conversations map (populated by get-conversations on
  // connect, so true for any conversation with messages in it).
  openConversationById: (conversationId: number) => void
  getConversationTitle: (conversationId: number) => string
  createGroupChat: (name: string, usernames: string[]) => void
  closeChat: (conversationId: number) => void
  sendMessage: (conversationId: number, text: string) => void
  loadMoreHistory: (conversationId: number) => void
  sendTyping: (conversationId: number) => void
  sendStopTyping: (conversationId: number) => void
}

const ChatContext = createContext<ChatContextValue | null>(null)

interface RawChatHistoryItem {
  id: number
  from: string
  message: string
  created_at: string
}

interface RawSentMessage {
  id: number
  conversation_id: number
  from: string
  message: string
  sent: string
}

interface RawTypingEvent {
  conversation_id: number
  from: string
}

interface RawReadReceipt {
  conversation_id: number
  username: string
  message_id: number
}

interface RawMessageAck {
  conversation_id: number
  id: number
}

function otherMember(info: ConversationInfo, myUsername: string): string {
  return info.members.find((m) => m.username !== myUsername)?.username ?? info.name ?? 'Unknown'
}

function titleFor(info: ConversationInfo, myUsername: string): string {
  return info.is_group ? (info.name ?? 'Group chat') : otherMember(info, myUsername)
}

function readStatesFor(info: ConversationInfo, myUsername: string): Record<string, number> {
  const states: Record<string, number> = {}
  for (const state of info.read_states) {
    if (state.username !== myUsername) states[state.username] = state.last_read_message_id
  }
  return states
}

export function ChatProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const { subscribe, send } = useWebSocket()
  const { showMessage } = useStatusMessage()
  const myUsername = user?.username ?? ''

  // Asked for once per logged-in session, not per message — repeatedly
  // calling requestPermission on an already-decided (granted or denied)
  // permission is a harmless no-op, but there's no reason to try more than
  // once per mount.
  useEffect(() => {
    if (myUsername) requestNotificationPermission()
  }, [myUsername])

  const [onlineUsernames, setOnlineUsernames] = useState<string[]>([])
  const [unreadConversations, setUnreadConversations] = useState<Set<number>>(new Set())
  const [openWindows, setOpenWindows] = useState<Record<number, ChatWindowState>>({})
  const [conversations, setConversations] = useState<Record<number, ConversationInfo>>({})

  // Subscription handlers close over these to read current state without
  // needing to resubscribe every time it changes.
  const openWindowsRef = useRef(openWindows)
  useEffect(() => {
    openWindowsRef.current = openWindows
  }, [openWindows])
  const conversationsRef = useRef(conversations)
  useEffect(() => {
    conversationsRef.current = conversations
  }, [conversations])
  // FIFO correlation for get-chat-history/get-more-chat-history requests —
  // the chat_history response carries no conversation_id of its own when
  // it's empty (nothing to infer it from), and there's no request/response
  // id in this WS protocol. Requests are sent and answered in order over a
  // single connection, so a queue is a safe enough correlation mechanism.
  const pendingHistoryRef = useRef<number[]>([])

  const directConversationIdByUsername = useMemo(() => {
    const map: Record<string, number> = {}
    for (const info of Object.values(conversations)) {
      if (!info.is_group) {
        map[otherMember(info, myUsername)] = info.conversation_id
      }
    }
    return map
  }, [conversations, myUsername])

  const groupChats = useMemo(() => Object.values(conversations).filter((c) => c.is_group), [conversations])

  const getConversationTitle = useCallback(
    (conversationId: number) => {
      const info = conversations[conversationId]
      return info ? titleFor(info, myUsername) : `Conversation ${conversationId}`
    },
    [conversations, myUsername],
  )

  const markUnread = useCallback((conversationId: number) => {
    setUnreadConversations((prev) => (prev.has(conversationId) ? prev : new Set(prev).add(conversationId)))
  }, [])

  // Ensures a window exists for a conversation (fetching page-1 history the
  // first time), and clears its unread state — the single place both
  // "I clicked to open this" and "the server just pushed this at me"
  // (chat-opened) converge.
  const revealWindow = useCallback(
    (info: ConversationInfo) => {
      setUnreadConversations((prev) => {
        if (!prev.has(info.conversation_id)) return prev
        const next = new Set(prev)
        next.delete(info.conversation_id)
        return next
      })
      if (!openWindowsRef.current[info.conversation_id]) {
        setOpenWindows((prev) => ({
          ...prev,
          [info.conversation_id]: {
            conversationId: info.conversation_id,
            isGroup: info.is_group,
            title: titleFor(info, myUsername),
            messages: [],
            offset: 0,
            loadingHistory: true,
            typingUsers: new Set(),
            readStates: readStatesFor(info, myUsername),
          },
        }))
        pendingHistoryRef.current.push(info.conversation_id)
        send('get-chat-history', { conversation_id: info.conversation_id, offset: 0, limit: HISTORY_PAGE_SIZE })
      }
    },
    [send, myUsername],
  )
  const revealWindowRef = useRef(revealWindow)
  useEffect(() => {
    revealWindowRef.current = revealWindow
  }, [revealWindow])

  const openConversationById = useCallback(
    (conversationId: number) => {
      const info = conversationsRef.current[conversationId]
      if (info) revealWindow(info)
    },
    [revealWindow],
  )

  // Reports having seen up to messageId — skipped for the sender's own
  // messages, since ws-manager.go's sendMessage already auto-advances the
  // sender's own watermark server-side.
  const markRead = useCallback(
    (conversationId: number, messageId: number) => {
      if (messageId > 0) send('mark-read', { conversation_id: conversationId, message_id: messageId })
    },
    [send],
  )

  useEffect(() => {
    const unsubUsers = subscribe('users-online', (payload) => {
      setOnlineUsernames(Object.keys(payload as Record<string, boolean>))
    })

    const unsubChatOpened = subscribe('chat-opened', (payload) => {
      const info = payload as ConversationInfo
      setConversations((prev) => ({ ...prev, [info.conversation_id]: info }))
      revealWindowRef.current(info)
    })

    const unsubConversationsList = subscribe('conversations-list', (payload) => {
      const list = payload as ConversationInfo[]
      setConversations((prev) => {
        const next = { ...prev }
        for (const info of list) next[info.conversation_id] = info
        return next
      })
    })

    const unsubSent = subscribe('sent-message', (payload) => {
      const data = payload as RawSentMessage
      const vm: ChatMessageVM = { id: data.id, from: data.from, message: data.message, timestamp: data.sent }

      // A direct conversation's very first incoming message can arrive
      // before this client ever learned about it (it only opened by the
      // *other* party calling open-direct-chat) — synthesize minimal
      // metadata from the message itself rather than requiring a round
      // trip just to find out who it's with. A group conversation is
      // always pushed via chat-opened at creation time to every then-online
      // member, so this fallback path is direct-chat-only.
      let info = conversationsRef.current[data.conversation_id]
      if (!info) {
        info = {
          conversation_id: data.conversation_id,
          is_group: false,
          members: [
            { user_id: -1, username: data.from },
            { user_id: -1, username: myUsername },
          ],
          read_states: [],
        }
        setConversations((prev) => ({ ...prev, [info.conversation_id]: info }))
      }

      if (openWindowsRef.current[data.conversation_id]) {
        setOpenWindows((prev) => ({
          ...prev,
          [data.conversation_id]: { ...prev[data.conversation_id], messages: [...prev[data.conversation_id].messages, vm] },
        }))
        // The window is open, so the user is presumably seeing this now.
        markRead(data.conversation_id, data.id)
      } else {
        markUnread(data.conversation_id)
        // Only for the "wasn't already open" case — if the window IS open,
        // the message already appears directly in the visible chat UI, so a
        // toast on top of it would just be noise.
        const title = info.is_group ? (info.name ?? 'Group chat') : data.from
        showMessage(`New message from ${title}: ${data.message}`, 'info')
      }

      // A native OS notification, independent of whether the window is
      // open — it's specifically for when the user isn't looking at this
      // tab at all, which the in-app toast above can't reach them through.
      if (isTabInBackground()) {
        const title = info.is_group ? (info.name ?? 'Group chat') : data.from
        const body = info.is_group ? `${data.from}: ${data.message}` : data.message
        showBrowserNotification(title, body)
      }
    })

    const unsubHistory = subscribe('chat_history', (payload) => {
      const convId = pendingHistoryRef.current.shift()
      if (convId === undefined) return
      const batch = (payload as RawChatHistoryItem[] | null) ?? []
      setOpenWindows((prev) => {
        const existing = prev[convId]
        if (!existing) return prev
        const vms: ChatMessageVM[] = batch.map((m) => ({ id: m.id, from: m.from, message: m.message, timestamp: m.created_at }))
        return { ...prev, [convId]: { ...existing, messages: [...vms, ...existing.messages], loadingHistory: false } }
      })
      if (batch.length > 0) {
        markRead(convId, Math.max(...batch.map((m) => m.id)))
      }
    })

    const unsubReadReceipt = subscribe('read-receipt', (payload) => {
      const data = payload as RawReadReceipt
      setOpenWindows((prev) => {
        const existing = prev[data.conversation_id]
        if (!existing) return prev
        return {
          ...prev,
          [data.conversation_id]: { ...existing, readStates: { ...existing.readStates, [data.username]: data.message_id } },
        }
      })
    })

    const unsubMessageAck = subscribe('message-ack', (payload) => {
      const data = payload as RawMessageAck
      // Reconciles this client's own just-sent message with its real,
      // database-assigned id — the server never echoes "sent-message" back
      // to its own sender, so without this the sender's own latest message
      // would carry id 0 forever and could never show a "seen by" state.
      // Matches the oldest unconfirmed (id: 0) entry, since sends from a
      // single client are acked in the order they were made.
      setOpenWindows((prev) => {
        const existing = prev[data.conversation_id]
        if (!existing) return prev
        const index = existing.messages.findIndex((m) => m.id === 0)
        if (index === -1) return prev
        const messages = [...existing.messages]
        messages[index] = { ...messages[index], id: data.id }
        return { ...prev, [data.conversation_id]: { ...existing, messages } }
      })
    })

    const unsubTyping = subscribe('typing', (payload) => {
      const data = payload as RawTypingEvent
      setOpenWindows((prev) => {
        const existing = prev[data.conversation_id]
        if (!existing) return prev
        return { ...prev, [data.conversation_id]: { ...existing, typingUsers: new Set(existing.typingUsers).add(data.from) } }
      })
    })

    const unsubStopTyping = subscribe('stop-typing', (payload) => {
      const data = payload as RawTypingEvent
      setOpenWindows((prev) => {
        const existing = prev[data.conversation_id]
        if (!existing) return prev
        const next = new Set(existing.typingUsers)
        next.delete(data.from)
        return { ...prev, [data.conversation_id]: { ...existing, typingUsers: next } }
      })
    })

    return () => {
      unsubUsers()
      unsubChatOpened()
      unsubConversationsList()
      unsubSent()
      unsubHistory()
      unsubReadReceipt()
      unsubMessageAck()
      unsubTyping()
      unsubStopTyping()
    }
  }, [subscribe, myUsername, markUnread, markRead, showMessage])

  const openDirectChat = useCallback(
    (username: string) => {
      if (username === myUsername) return
      const knownConvId = directConversationIdByUsername[username]
      const known = knownConvId !== undefined ? conversationsRef.current[knownConvId] : undefined
      if (known) {
        revealWindow(known)
        return
      }
      send('open-direct-chat', { username })
    },
    [send, myUsername, directConversationIdByUsername, revealWindow],
  )

  const createGroupChat = useCallback(
    (name: string, usernames: string[]) => {
      send('create-group-chat', { name, usernames })
    },
    [send],
  )

  const closeChat = useCallback((conversationId: number) => {
    setOpenWindows((prev) => {
      if (!(conversationId in prev)) return prev
      const next = { ...prev }
      delete next[conversationId]
      return next
    })
  }, [])

  const sendMessage = useCallback(
    (conversationId: number, text: string) => {
      // The server never echoes a sent message back to its own sender (see
      // ws-manager.go's sendMessage — it only broadcasts to other members)
      // — append it locally, optimistically, or the sender would never see
      // their own message in the window at all.
      // id: 0 marks this as an unconfirmed local echo — the server never
      // echoes a sent message back to its own sender, so this client never
      // learns this particular message's real id (its read watermark was
      // already auto-advanced server-side regardless, see sendMessage in
      // ws-manager.go).
      const vm: ChatMessageVM = { id: 0, from: myUsername, message: text, timestamp: Date.now() }
      setOpenWindows((prev) => {
        const existing = prev[conversationId]
        if (!existing) return prev
        return { ...prev, [conversationId]: { ...existing, messages: [...existing.messages, vm] } }
      })
      send('new-message', { conversation_id: conversationId, message: text })
    },
    [send, myUsername],
  )

  const loadMoreHistory = useCallback(
    (conversationId: number) => {
      setOpenWindows((prev) => {
        const existing = prev[conversationId]
        if (!existing || existing.loadingHistory) return prev
        const nextOffset = existing.offset + HISTORY_PAGE_SIZE
        pendingHistoryRef.current.push(conversationId)
        send('get-more-chat-history', { conversation_id: conversationId, offset: nextOffset, limit: HISTORY_PAGE_SIZE })
        return { ...prev, [conversationId]: { ...existing, offset: nextOffset, loadingHistory: true } }
      })
    },
    [send],
  )

  const sendTyping = useCallback((conversationId: number) => send('typing', { conversation_id: conversationId }), [send])
  const sendStopTyping = useCallback(
    (conversationId: number) => send('stop-typing', { conversation_id: conversationId }),
    [send],
  )

  const isUnreadUsername = useCallback(
    (username: string) => {
      const convId = directConversationIdByUsername[username]
      return convId !== undefined && unreadConversations.has(convId)
    },
    [directConversationIdByUsername, unreadConversations],
  )

  const unreadUsernames = useMemo(
    () => new Set(Object.keys(directConversationIdByUsername).filter(isUnreadUsername)),
    [directConversationIdByUsername, isUnreadUsername],
  )

  const onlineUsers = useMemo(() => {
    const unread = onlineUsernames.filter(isUnreadUsername)
    const rest = onlineUsernames.filter((u) => !isUnreadUsername(u)).sort()
    return [...unread, ...rest]
  }, [onlineUsernames, isUnreadUsername])

  return (
    <ChatContext.Provider
      value={{
        onlineUsers,
        unreadConversations,
        unreadUsernames,
        openWindows,
        groupChats,
        openDirectChat,
        openConversation: revealWindow,
        openConversationById,
        getConversationTitle,
        createGroupChat,
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

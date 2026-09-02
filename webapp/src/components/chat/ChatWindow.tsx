import { useRef, useState, type KeyboardEvent, type UIEvent } from 'react'
import { useChat, type ChatWindowState } from '../../contexts/ChatContext'
import type { ChatMessageVM } from '../../types'

function formatTimestamp(timestamp: string | number): string {
  const date = new Date(timestamp)
  return `${date.toLocaleDateString()}-${date.toLocaleTimeString()}`
}

// Loading older history shifts scroll position back to (or near) 0 once the
// new messages are prepended, and browsers keep firing 'scroll' events
// while pinned at the top. Without a cooldown, that resends
// get-more-chat-history — with an ever-incrementing offset — on every one
// of those events instead of once per deliberate scroll-to-top. There's no
// request/response id in this WS protocol to correlate a reply back to a
// specific request, so a short cooldown after sending is used instead of
// waiting for one.
const SCROLL_LOAD_COOLDOWN_MS = 800
const TYPING_IDLE_MS = 1000

interface ChatWindowProps {
  state: ChatWindowState
}

export function ChatWindow({ state }: ChatWindowProps) {
  const { closeChat, sendMessage, loadMoreHistory, sendTyping, sendStopTyping } = useChat()
  const [draft, setDraft] = useState('')
  const scrollCooldownRef = useRef(false)
  const typingTimeoutRef = useRef<number | undefined>(undefined)
  const { conversationId } = state

  function handleSend() {
    const text = draft.trim()
    if (!text) return
    sendMessage(conversationId, text)
    setDraft('')
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      handleSend()
    }
  }

  function handleInput(value: string) {
    setDraft(value)
    window.clearTimeout(typingTimeoutRef.current)
    sendTyping(conversationId)
    typingTimeoutRef.current = window.setTimeout(() => sendStopTyping(conversationId), TYPING_IDLE_MS)
  }

  function handleScroll(event: UIEvent<HTMLDivElement>) {
    const el = event.currentTarget
    if (el.scrollTop === 0 && !scrollCooldownRef.current) {
      scrollCooldownRef.current = true
      loadMoreHistory(conversationId)
      window.setTimeout(() => {
        scrollCooldownRef.current = false
      }, SCROLL_LOAD_COOLDOWN_MS)
    }
  }

  const typingLabel =
    state.typingUsers.size === 0
      ? null
      : state.isGroup
        ? `${[...state.typingUsers].join(', ')} typing...`
        : `${state.title} is typing...`

  // "Seen by" compares against the newest message with a real (server-
  // confirmed) id — id 0 marks a not-yet-reconciled local echo of this
  // client's own just-sent message, which can't be compared against a read
  // watermark yet.
  const latestConfirmedId = Math.max(0, ...state.messages.filter((m) => m.id > 0).map((m) => m.id))
  const seenBy =
    latestConfirmedId > 0
      ? Object.entries(state.readStates)
          .filter(([, lastRead]) => lastRead >= latestConfirmedId)
          .map(([username]) => username)
      : []

  return (
    <div className="chat-window" id={`chat:${conversationId}`}>
      <h3>{state.isGroup ? state.title : `Chat with ${state.title}`}</h3>
      <button type="button" className="close-chat" aria-label="Close chat" onClick={() => closeChat(conversationId)}>
        x
      </button>
      <div className="chat-messages" id={`chat-messages-${conversationId}`} onScroll={handleScroll}>
        <div className="spacer" style={{ height: 20 }} />
        {state.messages.map((message) => (
          // Not index-based: loadMoreHistory prepends whole batches to the
          // front, which would shift every later message's index and
          // confuse React's reconciliation. Real messages key off their
          // server id; an id-0 local echo (this client's own just-sent,
          // not-yet-reconciled message) falls back to a composite key.
          <ChatMessageRow
            key={message.id > 0 ? `msg-${message.id}` : `local-${message.from}-${message.timestamp}-${message.message}`}
            message={message}
          />
        ))}
      </div>
      {seenBy.length > 0 && <div className="seen-by">Seen by {seenBy.join(', ')}</div>}
      <div className="typing">
        {typingLabel && (
          <>
            <img
              id={`typing-indicator-${conversationId}`}
              src="/img/typing.gif"
              width={30}
              height={30}
              alt={state.isGroup ? '' : typingLabel}
            />
            {state.isGroup && <span>{typingLabel}</span>}
          </>
        )}
      </div>
      <div className="chat-footer">
        <textarea
          id={`new-message-${conversationId}`}
          placeholder="Type your message"
          value={draft}
          onChange={(event) => handleInput(event.target.value)}
          onKeyDown={handleKeyDown}
        />
        <button type="button" id={`message-submit-${conversationId}`} className="btns" onClick={handleSend}>
          Send
        </button>
      </div>
    </div>
  )
}

function ChatMessageRow({ message }: { message: ChatMessageVM }) {
  return (
    <p>
      <strong>
        {message.from} ({formatTimestamp(message.timestamp)}):{' '}
      </strong>
      {message.message}
    </p>
  )
}

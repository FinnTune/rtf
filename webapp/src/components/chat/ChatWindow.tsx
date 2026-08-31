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
  username: string
  state: ChatWindowState
}

export function ChatWindow({ username, state }: ChatWindowProps) {
  const { closeChat, sendMessage, loadMoreHistory, sendTyping, sendStopTyping } = useChat()
  const [draft, setDraft] = useState('')
  const scrollCooldownRef = useRef(false)
  const typingTimeoutRef = useRef<number | undefined>(undefined)

  function handleSend() {
    const text = draft.trim()
    if (!text) return
    sendMessage(username, text)
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
    sendTyping(username)
    typingTimeoutRef.current = window.setTimeout(() => sendStopTyping(username), TYPING_IDLE_MS)
  }

  function handleScroll(event: UIEvent<HTMLDivElement>) {
    const el = event.currentTarget
    if (el.scrollTop === 0 && !scrollCooldownRef.current) {
      scrollCooldownRef.current = true
      loadMoreHistory(username)
      window.setTimeout(() => {
        scrollCooldownRef.current = false
      }, SCROLL_LOAD_COOLDOWN_MS)
    }
  }

  return (
    <div className="chat-window" id={`chat:${username}`}>
      <h3>Chat with {username}</h3>
      <button type="button" className="close-chat" onClick={() => closeChat(username)}>
        x
      </button>
      <div className="chat-messages" id={`chat-messages-${username}`} onScroll={handleScroll}>
        <div className="spacer" style={{ height: 20 }} />
        {state.messages.map((message) => (
          // Not index-based: loadMoreHistory prepends whole batches to the
          // front, which would shift every later message's index and
          // confuse React's reconciliation.
          <ChatMessageRow key={`${message.from}-${message.timestamp}-${message.message}`} message={message} />
        ))}
      </div>
      <div className="typing">
        {state.isOtherTyping && <img id={`typing-indicator-${username}`} src="/img/typing.gif" width={30} height={30} alt="" />}
      </div>
      <div className="chat-footer">
        <textarea
          id={`new-message-${username}`}
          placeholder="Type your message"
          value={draft}
          onChange={(event) => handleInput(event.target.value)}
          onKeyDown={handleKeyDown}
        />
        <button type="button" id={`message-submit-${username}`} className="btns" onClick={handleSend}>
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

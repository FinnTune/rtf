import { useState, type FormEvent } from 'react'
import { onActivationKey } from '../../a11y'
import { useAuth } from '../../contexts/AuthContext'
import { useChat } from '../../contexts/ChatContext'

export function OnlineUsersList() {
  const { user } = useAuth()
  const { onlineUsers, unreadUsernames, openDirectChat } = useChat()
  const [targetUsername, setTargetUsername] = useState('')

  // This list only ever shows who's online right now, but openDirectChat
  // (and the server's open-direct-chat handler behind it) resolves any
  // real username, online or not — a returning conversation just waits for
  // its next chat-opened/history load. Without this, there was no way to
  // start a NEW conversation with someone who wasn't online at that exact
  // moment; an unresolvable username surfaces via the existing chat-error
  // handling (see ChatContext's subscribe('chat-error', ...)), so no
  // extra error handling is needed here.
  function handleStartChat(event: FormEvent) {
    event.preventDefault()
    const trimmed = targetUsername.trim()
    if (!trimmed || trimmed === user?.username) return
    openDirectChat(trimmed)
    setTargetUsername('')
  }

  return (
    <>
      <form className="start-chat-form" onSubmit={handleStartChat}>
        <input
          type="text"
          aria-label="Message someone by username"
          placeholder="Message someone by username…"
          value={targetUsername}
          onChange={(event) => setTargetUsername(event.target.value)}
        />
        <button type="submit" className="btns" disabled={!targetUsername.trim()}>
          Chat
        </button>
      </form>
      <ul id="users-list">
        {onlineUsers.map((username) => {
          function activate() {
            if (username !== user?.username) {
              openDirectChat(username)
            }
          }
          return (
            <li key={username} role="button" tabIndex={0} onClick={activate} onKeyDown={onActivationKey(activate)}>
              <span className="online-dot" aria-hidden="true" />
              {username}
              {unreadUsernames.has(username) && <span className="msg-alert" aria-label="Unread messages">!</span>}
            </li>
          )
        })}
      </ul>
    </>
  )
}

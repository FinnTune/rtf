import { useState } from 'react'
import { onActivationKey } from '../../a11y'
import { useChat } from '../../contexts/ChatContext'

export function GroupChatsPanel() {
  const { groupChats, unreadConversations, onlineUsers, createGroupChat, openConversation } = useChat()
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  function toggleMember(username: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(username)) next.delete(username)
      else next.add(username)
      return next
    })
  }

  function handleCreate() {
    const trimmedName = name.trim()
    if (!trimmedName || selected.size === 0) return
    createGroupChat(trimmedName, [...selected])
    setName('')
    setSelected(new Set())
    setCreating(false)
  }

  return (
    <div id="group-chats">
      <h3>
        Groups
        <button type="button" className="btns new-group-toggle" onClick={() => setCreating((prev) => !prev)}>
          {creating ? 'Cancel' : '+ New Group'}
        </button>
      </h3>

      {creating && (
        <div className="new-group-form">
          <input
            type="text"
            placeholder="Group name"
            aria-label="Group name"
            value={name}
            maxLength={50}
            onChange={(event) => setName(event.target.value)}
          />
          <p className="new-group-members-label">Add members (pick from who's online now):</p>
          <ul className="new-group-members">
            {onlineUsers.map((username) => (
              <li key={username}>
                <label>
                  <input type="checkbox" checked={selected.has(username)} onChange={() => toggleMember(username)} />
                  {username}
                </label>
              </li>
            ))}
          </ul>
          <button type="button" className="btns btn-primary" disabled={!name.trim() || selected.size === 0} onClick={handleCreate}>
            Create Group
          </button>
        </div>
      )}

      <ul id="group-chats-list">
        {groupChats.map((info) => (
          <li
            key={info.conversation_id}
            role="button"
            tabIndex={0}
            onClick={() => openConversation(info)}
            onKeyDown={onActivationKey(() => openConversation(info))}
          >
            {info.name}
            {unreadConversations.has(info.conversation_id) && <span className="msg-alert" aria-label="Unread messages">!</span>}
          </li>
        ))}
      </ul>
    </div>
  )
}

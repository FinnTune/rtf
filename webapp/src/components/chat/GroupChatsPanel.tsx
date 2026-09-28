import { useState, type FormEvent } from 'react'
import { onActivationKey } from '../../a11y'
import { useChat } from '../../contexts/ChatContext'

export function GroupChatsPanel() {
  const { groupChats, unreadConversations, onlineUsers, createGroupChat, openConversation } = useChat()
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [offlineUsername, setOfflineUsername] = useState('')

  function toggleMember(username: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(username)) next.delete(username)
      else next.add(username)
      return next
    })
  }

  // The checkbox list below only ever offers members who happen to be
  // online right now, but createGroupChat (and the server's
  // create-group-chat handler behind it) resolves any real username,
  // online or not — an unresolvable one surfaces via ChatContext's
  // existing chat-error handling, so nothing extra is needed here for
  // that case.
  function addOfflineMember(event: FormEvent) {
    event.preventDefault()
    const trimmed = offlineUsername.trim()
    if (!trimmed) return
    setSelected((prev) => new Set(prev).add(trimmed))
    setOfflineUsername('')
  }

  function removeMember(username: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      next.delete(username)
      return next
    })
  }

  function handleCreate() {
    const trimmedName = name.trim()
    if (!trimmedName || selected.size === 0) return
    createGroupChat(trimmedName, [...selected])
    setName('')
    setSelected(new Set())
    setOfflineUsername('')
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
          <p className="new-group-members-label">Add members who are online now:</p>
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
          <form className="add-offline-member-form" onSubmit={addOfflineMember}>
            <input
              type="text"
              aria-label="Add a member by username"
              placeholder="Add someone offline by username…"
              value={offlineUsername}
              onChange={(event) => setOfflineUsername(event.target.value)}
            />
            <button type="submit" className="btns" disabled={!offlineUsername.trim()}>
              Add
            </button>
          </form>
          {[...selected].filter((username) => !onlineUsers.includes(username)).length > 0 && (
            <ul className="new-group-offline-members">
              {[...selected]
                .filter((username) => !onlineUsers.includes(username))
                .map((username) => (
                  <li key={username}>
                    {username}
                    <button type="button" className="btns" aria-label={`Remove ${username}`} onClick={() => removeMember(username)}>
                      ×
                    </button>
                  </li>
                ))}
            </ul>
          )}
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

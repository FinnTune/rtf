import { onActivationKey } from '../../a11y'
import { useAuth } from '../../contexts/AuthContext'
import { useChat } from '../../contexts/ChatContext'

export function OnlineUsersList() {
  const { user } = useAuth()
  const { onlineUsers, unreadUsernames, openDirectChat } = useChat()

  return (
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
  )
}

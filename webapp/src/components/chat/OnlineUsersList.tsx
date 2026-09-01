import { useAuth } from '../../contexts/AuthContext'
import { useChat } from '../../contexts/ChatContext'

export function OnlineUsersList() {
  const { user } = useAuth()
  const { onlineUsers, unreadUsernames, openDirectChat } = useChat()

  return (
    <ul id="users-list">
      {onlineUsers.map((username) => (
        <li
          key={username}
          onClick={() => {
            if (username !== user?.username) {
              openDirectChat(username)
            }
          }}
        >
          <span className="online-dot" />
          {username}
          {unreadUsernames.has(username) && <span className="msg-alert">!</span>}
        </li>
      ))}
    </ul>
  )
}

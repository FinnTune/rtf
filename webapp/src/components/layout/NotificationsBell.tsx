import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useNotifications } from '../../contexts/NotificationsContext'

export function NotificationsBell() {
  const { notifications, unreadCount, markRead, markAllRead } = useNotifications()
  const [isOpen, setIsOpen] = useState(false)
  const navigate = useNavigate()

  function openNotification(id: number, postId: number) {
    markRead(id)
    setIsOpen(false)
    navigate(`/posts/${postId}`)
  }

  return (
    <div className="dropdown" id="notifications-bell">
      <button type="button" className="dropdown-toggle" aria-expanded={isOpen} onClick={() => setIsOpen((open) => !open)}>
        Notifications
        {unreadCount > 0 && (
          <span className="msg-alert" aria-label={`${unreadCount} unread notifications`}>
            {unreadCount}
          </span>
        )}
      </button>
      {isOpen && (
        <div className="dropdown-content">
          {notifications.length === 0 && <p className="empty-state">No notifications yet.</p>}
          {notifications.length > 0 && (
            <button type="button" className="btns" onClick={markAllRead} disabled={unreadCount === 0}>
              Mark all read
            </button>
          )}
          <ul className="notification-list">
            {notifications.map((n) => (
              <li key={n.id} className={n.read ? 'notification-item' : 'notification-item unread'}>
                <button type="button" onClick={() => openNotification(n.id, n.post_id)}>
                  <strong>{n.actor_username}</strong> commented on &quot;{n.post_title}&quot;
                  <span className="notification-time">{n.created_at}</span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}

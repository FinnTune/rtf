import { GroupChatsPanel } from '../chat/GroupChatsPanel'
import { OnlineUsersList } from '../chat/OnlineUsersList'

export function SidebarRight() {
  return (
    <aside className="sidebar-right">
      <div id="users">
        <h3>Users</h3>
        <OnlineUsersList />
      </div>
      <GroupChatsPanel />
    </aside>
  )
}

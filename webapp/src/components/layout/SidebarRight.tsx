export function SidebarRight() {
  return (
    <aside className="sidebar-right">
      <div id="users">
        <h3>Users</h3>
        {/* Populated once chat/presence lands in a later sub-PR. */}
        <ul id="users-list" />
      </div>
    </aside>
  )
}

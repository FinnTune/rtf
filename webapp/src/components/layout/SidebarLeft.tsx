import { NavLink } from 'react-router-dom'

function navItemClass({ isActive }: { isActive: boolean }) {
  return isActive ? 'nav-item active' : 'nav-item'
}

export function SidebarLeft() {
  return (
    <nav className="sidebar-left">
      <NavLink to="/" end className={navItemClass} id="all-posts-button">
        All Posts
      </NavLink>
      <NavLink to="/new-post" className={navItemClass} id="create-post-button">
        New Post
      </NavLink>
      <h4 className="sidebar-heading">Categories</h4>
      {/* Populated once category browsing lands in a later sub-PR. */}
      <div id="category-selection" className="category-nav" />
    </nav>
  )
}

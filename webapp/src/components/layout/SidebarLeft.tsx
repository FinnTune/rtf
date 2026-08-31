import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useFeedView } from '../../contexts/FeedViewContext'
import { CategoryNav } from '../posts/CategoryNav'

function navItemClass(active: boolean) {
  return active ? 'nav-item active' : 'nav-item'
}

export function SidebarLeft() {
  const { pathname } = useLocation()
  const { user } = useAuth()
  const { view, showAllPosts } = useFeedView()

  // "All Posts" is only the active nav item when we're both on the feed
  // route AND not currently viewing a category/search within it — a plain
  // route-based NavLink can't express the second half of that.
  const isAllPostsActive = pathname === '/' && view.type === 'all'
  const isNewPostActive = pathname === '/new-post'
  const isManageCategoriesActive = pathname === '/admin/categories'

  return (
    <nav className="sidebar-left">
      <Link to="/" className={navItemClass(isAllPostsActive)} id="all-posts-button" onClick={showAllPosts}>
        All Posts
      </Link>
      <Link to="/new-post" className={navItemClass(isNewPostActive)} id="create-post-button">
        New Post
      </Link>
      {user?.role === 'admin' && (
        <Link to="/admin/categories" className={navItemClass(isManageCategoriesActive)} id="manage-categories-button">
          Manage Categories
        </Link>
      )}
      <h4 className="sidebar-heading">Categories</h4>
      <div id="category-selection" className="category-nav">
        <CategoryNav />
      </div>
    </nav>
  )
}

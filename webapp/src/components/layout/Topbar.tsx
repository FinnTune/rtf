import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useFeedView } from '../../contexts/FeedViewContext'
import { SearchBox } from './SearchBox'

export function Topbar() {
  const { user, logout } = useAuth()
  const { view, showSearch, showAllPosts, searchResetToken } = useFeedView()
  const navigate = useNavigate()

  function handleSearchSubmit(query: string) {
    showSearch(query)
    navigate('/')
  }

  function handleClear() {
    showAllPosts()
    navigate('/')
  }

  return (
    <header className="topbar">
      <h1 id="title">
        <Link to="/" onClick={showAllPosts}>
          theDialectic
        </Link>
      </h1>
      <SearchBox key={searchResetToken} showClear={view.type === 'search'} onSubmit={handleSearchSubmit} onClear={handleClear} />
      <div className="topbar-user">
        <span id="topbar-username">{user?.username}</span>
        <button type="button" className="header-btns" id="logout-button" onClick={() => void logout()}>
          Logout
        </button>
      </div>
    </header>
  )
}

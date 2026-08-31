import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'

export function Topbar() {
  const { user, logout } = useAuth()
  const [query, setQuery] = useState('')

  // Search itself lands with the rest of the post list in a later sub-PR —
  // this just holds the input's place in the persistent shell for now.
  function handleSearchSubmit(event: FormEvent) {
    event.preventDefault()
  }

  return (
    <header className="topbar">
      <h1 id="title">
        <Link to="/">theDialectic</Link>
      </h1>
      <form id="search-form" className="topbar-search" onSubmit={handleSearchSubmit}>
        <input
          type="text"
          id="search-input"
          name="q"
          placeholder="Search posts..."
          maxLength={100}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <button type="submit" className="btns" id="search-submit-button">
          Search
        </button>
      </form>
      <div className="topbar-user">
        <span id="topbar-username">{user?.username}</span>
        <button type="button" className="header-btns" id="logout-button" onClick={() => void logout()}>
          Logout
        </button>
      </div>
    </header>
  )
}

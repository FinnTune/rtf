import { Route, Routes } from 'react-router-dom'
import { AuthorPostsPage } from './components/posts/AuthorPostsPage'
import { Feed } from './components/posts/Feed'
import { useAuth } from './contexts/AuthContext'
import { LoggedInShell } from './components/layout/LoggedInShell'
import { LoggedOutShell } from './components/layout/LoggedOutShell'

// Real views for these two land in sub-PR 4b (single post + comments,
// add/edit post).
function Placeholder({ label }: { label: string }) {
  return <p>{label} — coming soon.</p>
}

export default function App() {
  const { user, loading } = useAuth()

  if (loading) {
    return <p>Loading…</p>
  }

  if (!user) {
    // Deliberately path-agnostic: a logged-out visitor sees the same
    // landing page regardless of URL (e.g. a shared /posts/:id link),
    // matching the original app's behavior.
    return <LoggedOutShell />
  }

  return (
    <LoggedInShell>
      <Routes>
        <Route path="/" element={<Feed />} />
        <Route path="/posts/:id" element={<Placeholder label="Post" />} />
        <Route path="/users/:username" element={<AuthorPostsPage />} />
        <Route path="/new-post" element={<Placeholder label="New post" />} />
      </Routes>
    </LoggedInShell>
  )
}

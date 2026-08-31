import { Route, Routes } from 'react-router-dom'
import { AddPostForm } from './components/posts/AddPostForm'
import { AuthorPostsPage } from './components/posts/AuthorPostsPage'
import { Feed } from './components/posts/Feed'
import { SinglePostView } from './components/posts/SinglePostView'
import { ManageCategoriesPage } from './components/admin/ManageCategoriesPage'
import { useAuth } from './contexts/AuthContext'
import { LoggedInShell } from './components/layout/LoggedInShell'
import { LoggedOutShell } from './components/layout/LoggedOutShell'

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
        <Route path="/posts/:id" element={<SinglePostView />} />
        <Route path="/users/:username" element={<AuthorPostsPage />} />
        <Route path="/new-post" element={<AddPostForm />} />
        <Route path="/admin/categories" element={<ManageCategoriesPage />} />
      </Routes>
    </LoggedInShell>
  )
}

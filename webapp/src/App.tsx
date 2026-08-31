import { Route, Routes } from 'react-router-dom'
import { useAuth } from './contexts/AuthContext'
import { LoggedInShell } from './components/layout/LoggedInShell'
import { LoggedOutShell } from './components/layout/LoggedOutShell'

// Real routes/views land in later sub-PRs (see the migration plan) — this
// placeholder just proves the shell/routing/auth-state pipeline works
// end-to-end.
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
        <Route path="/" element={<Placeholder label="Feed" />} />
        <Route path="/posts/:id" element={<Placeholder label="Post" />} />
        <Route path="/users/:username" element={<Placeholder label="User posts" />} />
        <Route path="/new-post" element={<Placeholder label="New post" />} />
      </Routes>
    </LoggedInShell>
  )
}

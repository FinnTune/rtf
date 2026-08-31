import { Route, Routes } from 'react-router-dom'
import { useAuth } from './contexts/AuthContext'

// Scaffold placeholder only — real views land in later sub-PRs (see the
// migration plan). This just proves the routing/auth-state pipeline works
// end-to-end before any real feature code is written.
function Placeholder({ label }: { label: string }) {
  return <p>{label} — coming soon.</p>
}

export default function App() {
  const { user, loading } = useAuth()

  if (loading) {
    return <p>Loading…</p>
  }

  return (
    <div>
      <p>{user ? `Logged in as ${user.username}` : 'Not logged in'}</p>
      <Routes>
        <Route path="/" element={<Placeholder label="Feed" />} />
        <Route path="/posts/:id" element={<Placeholder label="Post" />} />
        <Route path="/users/:username" element={<Placeholder label="User posts" />} />
        <Route path="/new-post" element={<Placeholder label="New post" />} />
      </Routes>
    </div>
  )
}

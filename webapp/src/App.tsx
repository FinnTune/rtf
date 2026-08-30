import { Route, Routes } from 'react-router-dom'

// Scaffold placeholder only — real views land in later sub-PRs (see the
// migration plan). This just proves the build/routing/CI pipeline works
// end-to-end before any real feature code is written.
function Placeholder({ label }: { label: string }) {
  return <p>{label} — coming soon.</p>
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Placeholder label="Feed" />} />
      <Route path="/posts/:id" element={<Placeholder label="Post" />} />
      <Route path="/users/:username" element={<Placeholder label="User posts" />} />
      <Route path="/new-post" element={<Placeholder label="New post" />} />
    </Routes>
  )
}

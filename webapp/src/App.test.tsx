import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { AuthProvider } from './contexts/AuthContext'

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthProvider>
        <App />
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('App routing', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ loggedIn: false }), { status: 200 })),
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders the feed placeholder at /, once auth state resolves', async () => {
    renderAt('/')
    expect(await screen.findByText(/Feed — coming soon/)).toBeInTheDocument()
    expect(screen.getByText('Not logged in')).toBeInTheDocument()
  })

  it('renders the single-post placeholder at /posts/:id', async () => {
    renderAt('/posts/42')
    expect(await screen.findByText(/Post — coming soon/)).toBeInTheDocument()
  })
})

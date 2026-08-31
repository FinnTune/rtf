import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { AuthProvider } from './contexts/AuthContext'
import { StatusMessageProvider } from './contexts/StatusMessageContext'

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <StatusMessageProvider>
        <AuthProvider>
          <App />
        </AuthProvider>
      </StatusMessageProvider>
    </MemoryRouter>,
  )
}

function mockCheckLogin(response: unknown) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(response), { status: 200 })))
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('App', () => {
  it('shows the logged-out landing page when there is no session', async () => {
    mockCheckLogin({ loggedIn: false })
    renderAt('/')
    expect(await screen.findByText('Welcome to theDialectic')).toBeInTheDocument()
  })

  it('shows the logged-in shell and feed placeholder once /checkLogin confirms a session', async () => {
    mockCheckLogin({
      loggedIn: true,
      id: 1,
      username: 'alice',
      email: 'alice@example.com',
      joined: '2026-01-01',
      otp: 'otp-1',
    })
    renderAt('/')
    expect(await screen.findByText(/Feed — coming soon/)).toBeInTheDocument()
    expect(screen.getByText('alice')).toBeInTheDocument()
  })

  it('renders the logged-out landing page even at a deep-linked post URL when there is no session', async () => {
    mockCheckLogin({ loggedIn: false })
    renderAt('/posts/42')
    expect(await screen.findByText('Welcome to theDialectic')).toBeInTheDocument()
  })

  it('renders the single-post placeholder at /posts/:id once logged in', async () => {
    mockCheckLogin({
      loggedIn: true,
      id: 1,
      username: 'alice',
      email: 'alice@example.com',
      joined: '2026-01-01',
      otp: 'otp-1',
    })
    renderAt('/posts/42')
    expect(await screen.findByText(/Post — coming soon/)).toBeInTheDocument()
  })
})

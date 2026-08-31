import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { AuthProvider } from './contexts/AuthContext'
import { FeedViewProvider } from './contexts/FeedViewContext'
import { StatusMessageProvider } from './contexts/StatusMessageContext'

const loggedInCheckLoginBody = {
  loggedIn: true,
  id: 1,
  username: 'alice',
  email: 'alice@example.com',
  joined: '2026-01-01',
  otp: 'otp-1',
}

function requestUrl(input: string | URL | Request): string {
  if (typeof input === 'string') return input
  if (input instanceof URL) return input.toString()
  return input.url
}

// Routes each fetch by URL prefix, since App now renders real data-fetching
// components (Feed, CategoryNav), not just static placeholders.
function mockBackend(loggedIn: boolean) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) {
        return new Response(JSON.stringify(loggedIn ? loggedInCheckLoginBody : { loggedIn: false }), { status: 200 })
      }
      if (url.startsWith('/getAllPosts') || url.startsWith('/getPostsByAuthor')) {
        return new Response(JSON.stringify([]), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      if (url.startsWith('/getCategories')) {
        return new Response(JSON.stringify([]), { status: 200 })
      }
      if (url.startsWith('/getPost?')) {
        return new Response(
          JSON.stringify({ PostId: 42, UserId: 1, Title: 'Hello World', Content: 'Post body', Author: 'admin', Created: '2026-01-01' }),
          { status: 200 },
        )
      }
      if (url.startsWith('/comments')) {
        return new Response(JSON.stringify([]), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      throw new Error('Unexpected fetch in test: ' + url)
    }),
  )
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <StatusMessageProvider>
        <AuthProvider>
          <FeedViewProvider>
            <App />
          </FeedViewProvider>
        </AuthProvider>
      </StatusMessageProvider>
    </MemoryRouter>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('App', () => {
  it('shows the logged-out landing page when there is no session', async () => {
    mockBackend(false)
    renderAt('/')
    expect(await screen.findByText('Welcome to theDialectic')).toBeInTheDocument()
  })

  it('shows the logged-in shell and the real feed once /checkLogin confirms a session', async () => {
    mockBackend(true)
    renderAt('/')
    expect(await screen.findByText('No posts yet — be the first to post!')).toBeInTheDocument()
    expect(screen.getByText('alice')).toBeInTheDocument()
  })

  it('renders the logged-out landing page even at a deep-linked post URL when there is no session', async () => {
    mockBackend(false)
    renderAt('/posts/42')
    expect(await screen.findByText('Welcome to theDialectic')).toBeInTheDocument()
  })

  it('renders the real single-post view at /posts/:id once logged in', async () => {
    mockBackend(true)
    renderAt('/posts/42')
    expect(await screen.findByText('Hello World')).toBeInTheDocument()
    expect(screen.getByText('Post body')).toBeInTheDocument()
  })

  it('renders the add-post form at /new-post once logged in', async () => {
    mockBackend(true)
    renderAt('/new-post')
    expect(await screen.findByText('Add Post')).toBeInTheDocument()
  })

  it('renders the author-posts page at /users/:username once logged in', async () => {
    mockBackend(true)
    renderAt('/users/bob')
    expect(await screen.findByText('Posts by bob')).toBeInTheDocument()
    expect(await screen.findByText("bob hasn't posted yet.")).toBeInTheDocument()
  })
})

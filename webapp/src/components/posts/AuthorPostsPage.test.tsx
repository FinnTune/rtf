import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { AuthorPostsPage } from './AuthorPostsPage'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function mockBackend(profileStatus: number, profileBody: object | string = {}) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/getPostsByAuthor')) {
        return new Response(JSON.stringify([]), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      if (url.startsWith('/getUserProfile')) {
        const body = typeof profileBody === 'string' ? profileBody : JSON.stringify(profileBody)
        return new Response(body, { status: profileStatus })
      }
      throw new Error('Unexpected fetch: ' + url)
    }),
  )
}

function renderAt(username: string) {
  return render(
    <MemoryRouter initialEntries={[`/users/${username}`]}>
      <StatusMessageProvider>
        <Routes>
          <Route path="/users/:username" element={<AuthorPostsPage />} />
        </Routes>
      </StatusMessageProvider>
    </MemoryRouter>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('AuthorPostsPage', () => {
  it('shows the join date and pluralized post/comment counts once loaded', async () => {
    mockBackend(200, { username: 'bob', joined: '2026-01-01', post_count: 3, comment_count: 5 })
    renderAt('bob')

    expect(await screen.findByText('Joined 2026-01-01 · 3 posts · 5 comments')).toBeInTheDocument()
  })

  it('singularizes "1 post" / "1 comment" rather than "1 posts" / "1 comments"', async () => {
    mockBackend(200, { username: 'bob', joined: '2026-01-01', post_count: 1, comment_count: 1 })
    renderAt('bob')

    expect(await screen.findByText('Joined 2026-01-01 · 1 post · 1 comment')).toBeInTheDocument()
  })

  it('shows "User not found." instead of the post list when the profile fetch 404s', async () => {
    mockBackend(404, 'User not found')
    renderAt('nonexistent')

    expect(await screen.findByText('User not found.')).toBeInTheDocument()
    expect(screen.queryByText(/Posts by/)).not.toBeInTheDocument()
  })
})

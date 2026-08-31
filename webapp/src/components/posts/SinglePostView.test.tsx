import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { FeedViewProvider } from '../../contexts/FeedViewContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { SinglePostView } from './SinglePostView'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function mockBackend(loggedInAs: string, postAuthor: string) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) {
        return new Response(
          JSON.stringify({ loggedIn: true, id: 1, username: loggedInAs, email: 'a@example.com', joined: '2026-01-01', otp: 'x' }),
          { status: 200 },
        )
      }
      if (url.startsWith('/getPost')) {
        return new Response(
          JSON.stringify({ PostId: 7, UserId: 9, Title: 'A Post', Content: 'Body text', Author: postAuthor, Created: '2026-01-01' }),
          { status: 200 },
        )
      }
      if (url.startsWith('/comments')) {
        return new Response(JSON.stringify([]), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      if (url.startsWith('/deletePost')) {
        return new Response('Post deleted', { status: 200 })
      }
      if (url.startsWith('/getAllPosts')) {
        return new Response(JSON.stringify([]), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      if (url.startsWith('/getCategories')) {
        return new Response(JSON.stringify([]), { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    }),
  )
}

function renderPostRoute() {
  return render(
    <MemoryRouter initialEntries={['/posts/7']}>
      <StatusMessageProvider>
        <AuthProvider>
          <FeedViewProvider>
            <Routes>
              <Route path="/" element={<p>Feed placeholder</p>} />
              <Route path="/posts/:id" element={<SinglePostView />} />
            </Routes>
          </FeedViewProvider>
        </AuthProvider>
      </StatusMessageProvider>
    </MemoryRouter>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('SinglePostView', () => {
  it('fetches and displays the post', async () => {
    mockBackend('alice', 'admin')
    renderPostRoute()
    expect(await screen.findByText('A Post')).toBeInTheDocument()
    expect(screen.getByText('Body text')).toBeInTheDocument()
    expect(screen.getByText('admin')).toBeInTheDocument()
  })

  it('shows Edit/Delete only when the logged-in user is the post’s author', async () => {
    mockBackend('alice', 'bob')
    renderPostRoute()
    await screen.findByText('A Post')
    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
  })

  it('shows Edit/Delete for the post’s own author', async () => {
    mockBackend('admin', 'admin')
    renderPostRoute()
    expect(await screen.findByRole('button', { name: 'Edit' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
  })

  it('deletes the post after confirmation and navigates back to the feed', async () => {
    mockBackend('admin', 'admin')
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderPostRoute()
    await userEvent.click(await screen.findByRole('button', { name: 'Delete' }))
    expect(await screen.findByText('Feed placeholder')).toBeInTheDocument()
  })

  it('toggling Edit shows the edit form instead of the read-only view', async () => {
    mockBackend('admin', 'admin')
    renderPostRoute()
    await userEvent.click(await screen.findByRole('button', { name: 'Edit' }))
    expect(screen.getByDisplayValue('A Post')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Body text')).toBeInTheDocument()
    expect(screen.queryByText('A Post')).not.toBeInTheDocument()
  })
})

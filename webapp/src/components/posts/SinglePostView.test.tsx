import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { FeedViewProvider } from '../../contexts/FeedViewContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { StatusBanner } from '../common/StatusBanner'
import { SinglePostView } from './SinglePostView'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function mockBackend(loggedInAs: string, postAuthor: string, imgUrl = '') {
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
          JSON.stringify({
            PostId: 7,
            UserId: 9,
            Title: 'A Post',
            Content: 'Body text',
            Author: postAuthor,
            Created: '2026-01-01',
            ImgURL: imgUrl,
          }),
          { status: 200 },
        )
      }
      if (url.startsWith('/comments')) {
        return new Response(JSON.stringify([]), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      if (url.startsWith('/deletePost')) {
        return new Response('Post deleted', { status: 200 })
      }
      if (url.startsWith('/uploadPostImage')) {
        return new Response(JSON.stringify({ img_url: '/uploads/posts/newimage.png' }), { status: 200 })
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
            <StatusBanner />
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

  it('displays the post image when one is set', async () => {
    mockBackend('alice', 'admin', '/uploads/posts/existing.png')
    renderPostRoute()
    const img = await screen.findByRole('img', { name: 'A Post' })
    expect(img).toHaveAttribute('src', '/uploads/posts/existing.png')
  })

  it('does not render an image element when the post has none', async () => {
    mockBackend('alice', 'admin')
    renderPostRoute()
    await screen.findByText('A Post')
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })

  it('uploading an image from the edit form updates the displayed image', async () => {
    mockBackend('admin', 'admin')
    renderPostRoute()
    await userEvent.click(await screen.findByRole('button', { name: 'Edit' }))

    const fileInput = screen.getByLabelText('Image') as HTMLInputElement
    const file = new File(['fake-image-bytes'], 'photo.png', { type: 'image/png' })
    await userEvent.upload(fileInput, file)

    await userEvent.click(screen.getByRole('button', { name: 'Upload Image' }))

    expect(await screen.findByText('Image updated.')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    const img = await screen.findByRole('img', { name: 'A Post' })
    expect(img).toHaveAttribute('src', '/uploads/posts/newimage.png')
  })
})

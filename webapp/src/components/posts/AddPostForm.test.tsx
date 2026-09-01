import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { FeedViewProvider } from '../../contexts/FeedViewContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { StatusBanner } from '../common/StatusBanner'
import { AddPostForm } from './AddPostForm'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function mockBackend() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/getCategories')) {
        return new Response(JSON.stringify([{ id: 1, name: 'Sports' }]), { status: 200 })
      }
      if (url.startsWith('/addPost')) {
        return new Response(JSON.stringify({ id: 7 }), { status: 200 })
      }
      if (url.startsWith('/getAllPosts')) {
        return new Response(JSON.stringify([]), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      if (url.startsWith('/uploadPostImage')) {
        return new Response(JSON.stringify({ img_url: '/uploads/posts/newimage.png' }), { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    }),
  )
}

function renderAddPostForm() {
  return render(
    <MemoryRouter initialEntries={['/new-post']}>
      <StatusMessageProvider>
        <FeedViewProvider>
          <StatusBanner />
          <Routes>
            <Route path="/" element={<p>Feed placeholder</p>} />
            <Route path="/new-post" element={<AddPostForm />} />
          </Routes>
        </FeedViewProvider>
      </StatusMessageProvider>
    </MemoryRouter>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('AddPostForm', () => {
  it('submits the title/content/categories and returns to the feed on success', async () => {
    mockBackend()
    renderAddPostForm()

    await userEvent.type(screen.getByLabelText('Title:'), 'My New Post')
    await userEvent.type(screen.getByLabelText('Content:'), 'Some interesting content.')
    await userEvent.click(screen.getByText('Select Categories>>'))
    await userEvent.click(await screen.findByLabelText('Sports'))
    await userEvent.click(screen.getByRole('button', { name: 'Submit Post' }))

    await waitFor(() => expect(screen.getByText('Feed placeholder')).toBeInTheDocument())
    expect(await screen.findByText('Your post was submitted.')).toBeInTheDocument()

    const fetchMock = vi.mocked(fetch)
    const addPostCall = fetchMock.mock.calls.find(([input]) => requestUrl(input as string | URL | Request).startsWith('/addPost'))
    expect(addPostCall).toBeDefined()
    const body = JSON.parse((addPostCall![1] as RequestInit).body as string)
    expect(body).toEqual({ title: 'My New Post', content: 'Some interesting content.', categories: [{ id: 1, name: 'Sports' }] })
  })

  it('does not submit when title/content are empty', async () => {
    mockBackend()
    renderAddPostForm()
    await userEvent.click(screen.getByRole('button', { name: 'Submit Post' }))
    const fetchMock = vi.mocked(fetch)
    expect(fetchMock.mock.calls.some(([input]) => requestUrl(input as string | URL | Request).startsWith('/addPost'))).toBe(false)
  })

  it('uploads the selected image to the newly created post', async () => {
    mockBackend()
    renderAddPostForm()

    await userEvent.type(screen.getByLabelText('Title:'), 'My New Post')
    await userEvent.type(screen.getByLabelText('Content:'), 'Some interesting content.')
    const file = new File(['fake-image-bytes'], 'photo.png', { type: 'image/png' })
    await userEvent.upload(screen.getByLabelText('Image (optional):'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Submit Post' }))

    await waitFor(() => expect(screen.getByText('Feed placeholder')).toBeInTheDocument())
    expect(await screen.findByText('Your post was submitted.')).toBeInTheDocument()

    const fetchMock = vi.mocked(fetch)
    const uploadCall = fetchMock.mock.calls.find(([input]) =>
      requestUrl(input as string | URL | Request).startsWith('/uploadPostImage'),
    )
    expect(uploadCall).toBeDefined()
    const formData = (uploadCall![1] as RequestInit).body as FormData
    expect(formData.get('post_id')).toBe('7')
    expect((formData.get('image') as File).name).toBe('photo.png')
  })
})

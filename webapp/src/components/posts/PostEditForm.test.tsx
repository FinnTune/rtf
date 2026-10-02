import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import type { Post } from '../../types'
import { StatusBanner } from '../common/StatusBanner'
import { PostEditForm } from './PostEditForm'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function mockBackend(overrides: { editPostStatus?: number; editPostBody?: string } = {}) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/getCategories')) {
        return new Response(JSON.stringify([{ id: 1, name: 'Sports' }, { id: 2, name: 'Tech' }]), { status: 200 })
      }
      if (url.startsWith('/getPostCategories')) {
        return new Response(JSON.stringify([{ id: 2, name: 'Tech' }]), { status: 200 })
      }
      if (url.startsWith('/editPost')) {
        if (overrides.editPostStatus && overrides.editPostStatus >= 400) {
          return new Response(overrides.editPostBody ?? 'Failed to update post', { status: overrides.editPostStatus })
        }
        return new Response(JSON.stringify({ title: 'Updated Title', content: 'Updated content.' }), { status: 200 })
      }
      if (url.startsWith('/uploadPostImage')) {
        return new Response(JSON.stringify({ img_url: '/uploads/posts/newimage.png' }), { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    }),
  )
}

function makePost(overrides: Partial<Post> = {}): Post {
  return {
    PostId: 7,
    UserId: 1,
    Title: 'Original Title',
    Content: 'Original content.',
    Author: 'alice',
    Created: '2026-01-01',
    ImgURL: '',
    LikeCount: 0,
    DislikeCount: 0,
    MyReaction: 'none',
    ...overrides,
  }
}

function renderForm(post: Post, handlers: { onSaved?: (u: { title: string; content: string }) => void; onImageUploaded?: (url: string) => void; onCancel?: () => void } = {}) {
  const onSaved = handlers.onSaved ?? vi.fn()
  const onImageUploaded = handlers.onImageUploaded ?? vi.fn()
  const onCancel = handlers.onCancel ?? vi.fn()
  render(
    <StatusMessageProvider>
      <StatusBanner />
      <PostEditForm post={post} onSaved={onSaved} onImageUploaded={onImageUploaded} onCancel={onCancel} />
    </StatusMessageProvider>,
  )
  return { onSaved, onImageUploaded, onCancel }
}

function findEditPostCall() {
  const fetchMock = vi.mocked(fetch)
  return fetchMock.mock.calls.find(([input]) => requestUrl(input as string | URL | Request).startsWith('/editPost'))
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('PostEditForm', () => {
  it("loads and pre-selects the post's existing categories, and includes them in the save request", async () => {
    mockBackend()
    renderForm(makePost())

    await userEvent.click(screen.getByText('Select Categories>>'))
    expect(await screen.findByLabelText('Tech')).toBeChecked()
    expect(screen.getByLabelText('Sports')).not.toBeChecked()

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(findEditPostCall()).toBeDefined())
    const body = JSON.parse((findEditPostCall()![1] as RequestInit).body as string)
    expect(body.categories).toEqual([{ id: 2, name: 'Tech' }])
  })

  it('does not save when the title trims to empty', async () => {
    mockBackend()
    renderForm(makePost())

    await userEvent.clear(screen.getByLabelText('Title'))
    await userEvent.type(screen.getByLabelText('Title'), '   ')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(findEditPostCall()).toBeUndefined()
  })

  it('saves trimmed title/content and calls onSaved on success', async () => {
    mockBackend()
    const { onSaved } = renderForm(makePost())

    await userEvent.clear(screen.getByLabelText('Title'))
    await userEvent.type(screen.getByLabelText('Title'), '  Updated Title  ')
    await userEvent.clear(screen.getByLabelText('Content'))
    await userEvent.type(screen.getByLabelText('Content'), '  Updated content.  ')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('Post updated.')).toBeInTheDocument()
    expect(onSaved).toHaveBeenCalledTimes(1)

    const body = JSON.parse((findEditPostCall()![1] as RequestInit).body as string)
    expect(body.title).toBe('Updated Title')
    expect(body.content).toBe('Updated content.')
  })

  it('shows an error, leaves onSaved uncalled, and lets Save be retried if the request fails', async () => {
    mockBackend({ editPostStatus: 403, editPostBody: 'You can only edit your own posts' })
    const { onSaved } = renderForm(makePost())

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Err: You can only edit your own posts')).toBeInTheDocument()
    expect(onSaved).not.toHaveBeenCalled()
    // submitting was reset to false on failure, unlike the success path
    // (where the form is expected to unmount via onSaved instead) — the
    // button must be clickable again, not stuck showing "Saving...".
    expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
  })

  it('uploads a new image independently of Save, without submitting the title/content', async () => {
    mockBackend()
    const { onImageUploaded } = renderForm(makePost())

    const file = new File(['fake-image-bytes'], 'photo.png', { type: 'image/png' })
    await userEvent.upload(screen.getByLabelText('Image'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Upload Image' }))

    expect(await screen.findByText('Image updated.')).toBeInTheDocument()
    expect(onImageUploaded).toHaveBeenCalledWith('/uploads/posts/newimage.png')
    expect(findEditPostCall()).toBeUndefined()

    const fetchMock = vi.mocked(fetch)
    const uploadCall = fetchMock.mock.calls.find(([input]) => requestUrl(input as string | URL | Request).startsWith('/uploadPostImage'))
    const formData = (uploadCall![1] as RequestInit).body as FormData
    expect(formData.get('post_id')).toBe('7')
    expect((formData.get('image') as File).name).toBe('photo.png')
  })

  it('labels the button Replace Image (not Upload Image) when the post already has one', () => {
    mockBackend()
    renderForm(makePost({ ImgURL: '/uploads/posts/existing.png' }))
    expect(screen.getByRole('button', { name: 'Replace Image' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Upload Image' })).not.toBeInTheDocument()
  })

  it('disables the image button until a file is chosen', async () => {
    mockBackend()
    renderForm(makePost())
    expect(screen.getByRole('button', { name: 'Upload Image' })).toBeDisabled()

    const file = new File(['fake-image-bytes'], 'photo.png', { type: 'image/png' })
    await userEvent.upload(screen.getByLabelText('Image'), file)
    expect(screen.getByRole('button', { name: 'Upload Image' })).toBeEnabled()
  })

  it('calls onCancel, without saving anything, when Cancel is clicked', async () => {
    mockBackend()
    const { onCancel } = renderForm(makePost())

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onCancel).toHaveBeenCalledTimes(1)
    expect(findEditPostCall()).toBeUndefined()
  })
})

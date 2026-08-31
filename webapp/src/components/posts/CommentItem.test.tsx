import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import type { Comment } from '../../types'
import { CommentItem } from './CommentItem'

const comment: Comment = {
  id: 1,
  user_id: 5,
  post_id: 10,
  username: 'alice',
  content: 'Nice post!',
  created_at: '2026-01-01',
}

function checkLoginResponse(username: string | null) {
  return new Response(
    JSON.stringify(
      username
        ? { loggedIn: true, id: 5, username, email: 'a@example.com', joined: '2026-01-01', otp: 'x' }
        : { loggedIn: false },
    ),
    { status: 200 },
  )
}

function renderCommentItem(loggedInAs: string | null, overrides: Partial<{ onDeleted: () => void; onEdited: (c: string) => void }> = {}) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(checkLoginResponse(loggedInAs)))
  return render(
    <StatusMessageProvider>
      <AuthProvider>
        <CommentItem comment={comment} onDeleted={overrides.onDeleted ?? vi.fn()} onEdited={overrides.onEdited ?? vi.fn()} />
      </AuthProvider>
    </StatusMessageProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('CommentItem', () => {
  it('shows no Edit/Delete controls for another user’s comment', async () => {
    renderCommentItem('bob')
    await waitFor(() => expect(screen.getByText(/alice: Nice post!/)).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
  })

  it('shows Edit/Delete controls for the comment’s own author', async () => {
    renderCommentItem('alice')
    expect(await screen.findByRole('button', { name: 'Edit' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
  })

  it('saves an edit and calls onEdited with the server-confirmed content', async () => {
    const onEdited = vi.fn()
    renderCommentItem('alice', { onEdited })
    await screen.findByRole('button', { name: 'Edit' })

    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.startsWith('/editComment')) {
        return new Response(JSON.stringify({ content: 'Updated!' }), { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    const input = screen.getByDisplayValue('Nice post!')
    await userEvent.clear(input)
    await userEvent.type(input, 'Updated!')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(onEdited).toHaveBeenCalledWith('Updated!'))
    expect(fetchMock).toHaveBeenCalledWith(
      '/editComment',
      expect.objectContaining({ body: JSON.stringify({ id: 1, content: 'Updated!' }) }),
    )
  })

  it('deletes only after the user confirms, then calls onDeleted', async () => {
    const onDeleted = vi.fn()
    renderCommentItem('alice', { onDeleted })
    await screen.findByRole('button', { name: 'Delete' })

    const fetchMock = vi.fn().mockResolvedValue(new Response('Comment deleted', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    vi.spyOn(window, 'confirm').mockReturnValue(false)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(fetchMock).not.toHaveBeenCalled()
    expect(onDeleted).not.toHaveBeenCalled()

    vi.spyOn(window, 'confirm').mockReturnValue(true)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(onDeleted).toHaveBeenCalledOnce())
  })
})

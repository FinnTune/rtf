import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import type { Comment } from '../../types'
import { StatusBanner } from '../common/StatusBanner'
import { CommentForm } from './CommentForm'

vi.mock('../../api/comments', () => ({ addComment: vi.fn() }))
import { addComment } from '../../api/comments'

function makeComment(overrides: Partial<Comment> = {}): Comment {
  return {
    id: 1,
    post_id: 42,
    user_id: 1,
    username: 'alice',
    content: 'Nice post!',
    created_at: '2026-01-01',
    ...overrides,
  }
}

function renderForm(onAdded = vi.fn()) {
  render(
    <StatusMessageProvider>
      <StatusBanner />
      <CommentForm postId={42} onAdded={onAdded} />
    </StatusMessageProvider>,
  )
  return { onAdded }
}

beforeEach(() => {
  vi.mocked(addComment).mockReset()
})

describe('CommentForm', () => {
  it('does not submit when the comment trims to empty', async () => {
    renderForm()
    await userEvent.type(screen.getByLabelText('Enter your comment here'), '   ')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Comment' }))
    expect(addComment).not.toHaveBeenCalled()
  })

  it('submits the trimmed content, calls onAdded, and clears the input on success', async () => {
    const comment = makeComment({ content: 'Nice post!' })
    vi.mocked(addComment).mockResolvedValue(comment)
    const { onAdded } = renderForm()

    await userEvent.type(screen.getByLabelText('Enter your comment here'), '  Nice post!  ')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Comment' }))

    expect(await screen.findByLabelText('Enter your comment here')).toHaveValue('')
    expect(addComment).toHaveBeenCalledWith(42, 'Nice post!')
    expect(onAdded).toHaveBeenCalledWith(comment)
  })

  it('shows an error and keeps the typed text if the request fails', async () => {
    vi.mocked(addComment).mockRejectedValue(new Error('Post not found'))
    const { onAdded } = renderForm()

    await userEvent.type(screen.getByLabelText('Enter your comment here'), 'Nice post!')
    await userEvent.click(screen.getByRole('button', { name: 'Submit Comment' }))

    expect(await screen.findByText('Err: Post not found')).toBeInTheDocument()
    expect(onAdded).not.toHaveBeenCalled()
    expect(screen.getByLabelText('Enter your comment here')).toHaveValue('Nice post!')
  })
})

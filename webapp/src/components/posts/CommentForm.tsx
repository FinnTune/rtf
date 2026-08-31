import { useState, type FormEvent } from 'react'
import { addComment } from '../../api/comments'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Comment } from '../../types'
import { LoadingButton } from '../common/LoadingButton'

export function CommentForm({ postId, onAdded }: { postId: number; onAdded: (comment: Comment) => void }) {
  const [content, setContent] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const { showMessage } = useStatusMessage()

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    const trimmed = content.trim()
    if (!trimmed) {
      return
    }
    setSubmitting(true)
    try {
      const comment = await addComment(postId, trimmed)
      onAdded(comment)
      setContent('')
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="text"
        name="comment"
        placeholder="Enter your comment here"
        maxLength={500}
        required
        value={content}
        onChange={(event) => setContent(event.target.value)}
      />
      <LoadingButton type="submit" className="btns btn-primary" loading={submitting} loadingText="Posting...">
        Submit Comment
      </LoadingButton>
    </form>
  )
}

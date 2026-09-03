import { useState } from 'react'
import { deleteComment, editComment } from '../../api/comments'
import { useAuth } from '../../contexts/AuthContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Comment } from '../../types'
import { LoadingButton } from '../common/LoadingButton'

interface CommentItemProps {
  comment: Comment
  onDeleted: () => void
  onEdited: (content: string) => void
}

export function CommentItem({ comment, onDeleted, onEdited }: CommentItemProps) {
  const { user } = useAuth()
  const { showMessage } = useStatusMessage()
  const [editing, setEditing] = useState(false)
  const [content, setContent] = useState(comment.content)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const isOwn = user?.username === comment.username
  // Admins can delete (but not silently rewrite) another user's comment —
  // moderation, not impersonation. The backend re-checks this regardless
  // of what the client claims (see DeleteCommentHandler's admin check).
  const canDelete = isOwn || user?.role === 'admin'

  async function handleSave() {
    const trimmed = content.trim()
    if (!trimmed) {
      return
    }
    setSaving(true)
    try {
      const updated = await editComment(comment.id, trimmed)
      onEdited(updated.content)
      setEditing(false)
      showMessage('Comment updated.', 'success')
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      setSaving(false)
    }
  }

  async function handleDelete() {
    if (!window.confirm('Delete this comment? This cannot be undone.')) {
      return
    }
    setDeleting(true)
    try {
      await deleteComment(comment.id)
      onDeleted()
      showMessage('Comment deleted.', 'success')
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      setDeleting(false)
    }
  }

  if (editing) {
    return (
      <p className="comment">
        <input
          type="text"
          aria-label="Edit comment"
          value={content}
          maxLength={500}
          required
          onChange={(event) => setContent(event.target.value)}
        />
        <LoadingButton type="button" className="btns btn-primary" loading={saving} loadingText="Saving..." onClick={() => void handleSave()}>
          Save
        </LoadingButton>
        <button
          type="button"
          className="btns"
          onClick={() => {
            setEditing(false)
            setContent(comment.content)
          }}
        >
          Cancel
        </button>
      </p>
    )
  }

  return (
    <p className="comment">
      <span>
        {comment.username}: {comment.content}
      </span>
      {canDelete && (
        <>
          {isOwn && (
            <button type="button" className="btns" onClick={() => setEditing(true)}>
              Edit
            </button>
          )}
          <LoadingButton type="button" className="btns btn-danger" loading={deleting} loadingText="Deleting..." onClick={() => void handleDelete()}>
            Delete
          </LoadingButton>
        </>
      )}
    </p>
  )
}

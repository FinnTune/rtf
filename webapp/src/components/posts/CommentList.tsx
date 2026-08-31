import { useCallback, useEffect, useState } from 'react'
import { getComments } from '../../api/comments'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Comment } from '../../types'
import { CommentForm } from './CommentForm'
import { CommentItem } from './CommentItem'

const COMMENTS_PAGE_SIZE = 20

export function CommentList({ postId }: { postId: number }) {
  const [comments, setComments] = useState<Comment[]>([])
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const [loadingMore, setLoadingMore] = useState(false)
  const [loadedOnce, setLoadedOnce] = useState(false)
  const { showMessage } = useStatusMessage()

  const loadMore = useCallback(
    async (targetOffset: number) => {
      setLoadingMore(true)
      try {
        const { comments: page, total: newTotal } = await getComments(postId, targetOffset, COMMENTS_PAGE_SIZE)
        setComments((prev) => [...prev, ...page])
        setOffset(targetOffset + page.length)
        setTotal(newTotal)
      } catch (error) {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      } finally {
        setLoadingMore(false)
        setLoadedOnce(true)
      }
    },
    [postId, showMessage],
  )

  useEffect(() => {
    setComments([])
    setOffset(0)
    setTotal(0)
    setLoadedOnce(false)
    void loadMore(0)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- load the first page once per postId, not on every offset/loadMore identity change
  }, [postId])

  function handleAdded(comment: Comment) {
    setComments((prev) => [...prev, comment])
    setOffset((prev) => prev + 1)
    setTotal((prev) => prev + 1)
  }

  function handleDeleted(id: number) {
    setComments((prev) => prev.filter((comment) => comment.id !== id))
    setTotal((prev) => Math.max(0, prev - 1))
  }

  function handleEdited(id: number, content: string) {
    setComments((prev) => prev.map((comment) => (comment.id === id ? { ...comment, content } : comment)))
  }

  return (
    <div id="comments-section">
      <h4>Comments:</h4>
      <div id="comments-list">
        {loadedOnce && comments.length === 0 && <p className="empty-state">No comments yet — be the first to comment!</p>}
        {comments.map((comment) => (
          <CommentItem
            key={comment.id}
            comment={comment}
            onDeleted={() => handleDeleted(comment.id)}
            onEdited={(content) => handleEdited(comment.id, content)}
          />
        ))}
      </div>
      {offset < total && (
        <button type="button" className="btns" disabled={loadingMore} onClick={() => void loadMore(offset)}>
          {loadingMore ? 'Loading...' : 'Load more comments'}
        </button>
      )}
      <CommentForm postId={postId} onAdded={handleAdded} />
    </div>
  )
}

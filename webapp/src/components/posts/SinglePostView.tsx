import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { deletePost, getPost } from '../../api/posts'
import { useAuth } from '../../contexts/AuthContext'
import { useFeedView } from '../../contexts/FeedViewContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import { useOptionalWebSocket } from '../../contexts/WebSocketContext'
import type { Post } from '../../types'
import { LoadingButton } from '../common/LoadingButton'
import { CommentList } from './CommentList'
import { PostEditForm } from './PostEditForm'
import { ReactionButtons } from './ReactionButtons'

export function SinglePostView() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()
  const { showAllPosts } = useFeedView()
  const { showMessage } = useStatusMessage()
  const ws = useOptionalWebSocket()

  const [post, setPost] = useState<Post | null>(null)
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)
  const [editing, setEditing] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    setLoading(true)
    setNotFound(false)
    setEditing(false)
    getPost(id)
      .then(setPost)
      .catch((error: unknown) => {
        setNotFound(true)
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
      .finally(() => setLoading(false))
  }, [id, showMessage])

  // Keeps this permalink live for another connected client's own edits
  // too — the server broadcasts post-edited (excluding the editing client
  // itself, which already applies it via PostEditForm's onSaved callback)
  // whenever anyone edits this post. Filters against the current post
  // state directly (rather than a postId dependency) so it stays correct
  // if a stale broadcast for a since-navigated-away-from post arrives.
  useEffect(() => {
    if (!ws) return
    return ws.subscribe('post-edited', (payload) => {
      const update = payload as { post_id: number; title: string; content: string }
      setPost((prev) => (prev && prev.PostId === update.post_id ? { ...prev, Title: update.title, Content: update.content } : prev))
    })
  }, [ws])

  // Same live-update family as post-edited above, for this post being
  // deleted entirely instead of just changed — the server broadcasts
  // post-deleted (excluding the deleting client, which already navigates
  // away locally via handleDelete's own success path) whenever anyone
  // deletes this post. Compares against the route param id (a stable
  // string, unlike post state, which this effect deliberately avoids
  // depending on to keep the subscription from churning on every edit)
  // rather than post.PostId.
  useEffect(() => {
    if (!ws) return
    return ws.subscribe('post-deleted', (payload) => {
      const update = payload as { post_id: number }
      if (String(update.post_id) !== id) return
      showMessage('This post was deleted.', 'info')
      backToPosts()
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- backToPosts/showAllPosts/navigate aren't memoized; re-subscribing on their identity churn (every render) would be pure waste, and id/ws are the only inputs that actually matter for this comparison
  }, [ws, id])

  function backToPosts() {
    showAllPosts()
    navigate('/')
  }

  async function handleDelete() {
    if (!post) return
    if (!window.confirm('Delete this post? This cannot be undone.')) {
      return
    }
    setDeleting(true)
    try {
      await deletePost(post.PostId)
      showMessage('Post deleted.', 'success')
      backToPosts()
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      setDeleting(false)
    }
  }

  if (loading) {
    return <p>Loading…</p>
  }
  if (notFound || !post) {
    return <p className="empty-state">Post not found.</p>
  }

  const isOwn = user?.username === post.Author
  // Admins can delete (but not silently rewrite) another user's post —
  // moderation, not impersonation. The backend re-checks this regardless
  // of what the client claims (see DeletePostHandler's admin check).
  const canDelete = isOwn || user?.role === 'admin'

  return (
    <div id="single-post">
      {!editing && (
        <>
          <h3>{post.Title}</h3>
          {post.ImgURL && <img src={post.ImgURL} alt={post.Title} className="post-image" />}
          <p>{post.Content}</p>
        </>
      )}
      {editing && (
        <PostEditForm
          post={post}
          onSaved={(updated) => {
            setPost({ ...post, Title: updated.title, Content: updated.content })
            setEditing(false)
          }}
          onImageUploaded={(imgUrl) => setPost({ ...post, ImgURL: imgUrl })}
          onCancel={() => setEditing(false)}
        />
      )}
      <ReactionButtons post={post} />
      <p>
        Author:{' '}
        <Link to={`/users/${encodeURIComponent(post.Author)}`} className="author-link">
          {post.Author}
        </Link>
      </p>
      <p>Created: {post.Created}</p>
      {canDelete && !editing && (
        <div id="post-actions">
          {isOwn && (
            <button type="button" className="btns" onClick={() => setEditing(true)}>
              Edit
            </button>
          )}
          <LoadingButton
            type="button"
            className="btns btn-danger"
            loading={deleting}
            loadingText="Deleting..."
            onClick={() => void handleDelete()}
          >
            Delete
          </LoadingButton>
        </div>
      )}
      <button type="button" className="btns" onClick={backToPosts}>
        Back to Posts
      </button>
      <CommentList postId={post.PostId} />
    </div>
  )
}

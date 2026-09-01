import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { deletePost, getPost } from '../../api/posts'
import { useAuth } from '../../contexts/AuthContext'
import { useFeedView } from '../../contexts/FeedViewContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
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

  return (
    <div id="single-post">
      {!editing && (
        <>
          <h3>{post.Title}</h3>
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
      {isOwn && !editing && (
        <div id="post-actions">
          <button type="button" className="btns" onClick={() => setEditing(true)}>
            Edit
          </button>
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

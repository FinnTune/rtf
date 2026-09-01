import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { addPost, uploadPostImage, type PostCategoryRef } from '../../api/posts'
import { useFeedView } from '../../contexts/FeedViewContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import { LoadingButton } from '../common/LoadingButton'
import { CategoryPicker } from './CategoryPicker'

export function AddPostForm() {
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [categories, setCategories] = useState<PostCategoryRef[]>([])
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const { showMessage } = useStatusMessage()
  const { showAllPosts } = useFeedView()
  const navigate = useNavigate()

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    const trimmedTitle = title.trim()
    const trimmedContent = content.trim()
    if (!trimmedTitle || !trimmedContent) {
      return
    }
    setSubmitting(true)
    try {
      const postId = await addPost(trimmedTitle, trimmedContent, categories)
      // Image upload is a separate request (multipart, needs an existing
      // post id) — the post itself is already created at this point, so a
      // failure here is reported distinctly rather than as a failed post.
      if (imageFile) {
        try {
          await uploadPostImage(postId, imageFile)
        } catch (error) {
          showMessage(
            'Your post was submitted, but the image failed to upload: ' + (error instanceof Error ? error.message : String(error)),
            'error',
          )
          showAllPosts()
          navigate('/')
          return
        }
      }
      showAllPosts()
      navigate('/')
      showMessage('Your post was submitted.', 'success')
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="add-post" id="add-post">
      <h3>Add Post</h3>
      <form onSubmit={handleSubmit}>
        <label htmlFor="post-title">Title:</label>
        <input
          type="text"
          id="post-title"
          maxLength={100}
          required
          value={title}
          onChange={(event) => setTitle(event.target.value)}
        />
        <label htmlFor="post-content">Content:</label>
        <textarea
          id="post-content"
          rows={4}
          cols={50}
          maxLength={2000}
          required
          value={content}
          onChange={(event) => setContent(event.target.value)}
        />
        <div id="categories">
          <CategoryPicker selected={categories} onChange={setCategories} />
        </div>
        <label htmlFor="post-image">Image (optional):</label>
        <input
          type="file"
          id="post-image"
          accept="image/png, image/jpeg, image/gif, image/webp"
          onChange={(event) => setImageFile(event.target.files?.[0] ?? null)}
        />
        <LoadingButton type="submit" id="add-post-submit" loading={submitting} loadingText="Posting...">
          Submit Post
        </LoadingButton>
      </form>
    </div>
  )
}

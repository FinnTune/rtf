import { useEffect, useState } from 'react'
import { getPostCategories } from '../../api/categories'
import { editPost, type PostCategoryRef } from '../../api/posts'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Post } from '../../types'
import { LoadingButton } from '../common/LoadingButton'
import { CategoryPicker } from './CategoryPicker'

interface PostEditFormProps {
  post: Post
  onSaved: (updated: { title: string; content: string }) => void
  onCancel: () => void
}

export function PostEditForm({ post, onSaved, onCancel }: PostEditFormProps) {
  const [title, setTitle] = useState(post.Title)
  const [content, setContent] = useState(post.Content)
  const [categories, setCategories] = useState<PostCategoryRef[]>([])
  const [submitting, setSubmitting] = useState(false)
  const { showMessage } = useStatusMessage()

  useEffect(() => {
    getPostCategories(post.PostId)
      .then((existing) => setCategories(existing.map((category) => ({ id: category.id, name: category.name }))))
      .catch((error: unknown) => {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
  }, [post.PostId, showMessage])

  async function handleSave() {
    const trimmedTitle = title.trim()
    const trimmedContent = content.trim()
    if (!trimmedTitle || !trimmedContent) {
      return
    }
    setSubmitting(true)
    try {
      const updated = await editPost(post.PostId, trimmedTitle, trimmedContent, categories)
      showMessage('Post updated.', 'success')
      onSaved(updated)
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      setSubmitting(false)
    }
  }

  return (
    <div id="edit-post-form">
      <input type="text" value={title} maxLength={100} required onChange={(event) => setTitle(event.target.value)} />
      <textarea value={content} maxLength={2000} rows={4} cols={50} required onChange={(event) => setContent(event.target.value)} />
      <CategoryPicker selected={categories} onChange={setCategories} />
      <LoadingButton
        type="button"
        className="btns btn-primary"
        loading={submitting}
        loadingText="Saving..."
        onClick={() => void handleSave()}
      >
        Save
      </LoadingButton>
      <button type="button" className="btns" onClick={onCancel}>
        Cancel
      </button>
    </div>
  )
}

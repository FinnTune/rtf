import { useEffect, useState } from 'react'
import { getPostCategories } from '../../api/categories'
import { editPost, uploadPostImage, type PostCategoryRef } from '../../api/posts'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Post } from '../../types'
import { LoadingButton } from '../common/LoadingButton'
import { CategoryPicker } from './CategoryPicker'

interface PostEditFormProps {
  post: Post
  onSaved: (updated: { title: string; content: string }) => void
  onImageUploaded: (imgUrl: string) => void
  onCancel: () => void
}

export function PostEditForm({ post, onSaved, onImageUploaded, onCancel }: PostEditFormProps) {
  const [title, setTitle] = useState(post.Title)
  const [content, setContent] = useState(post.Content)
  const [categories, setCategories] = useState<PostCategoryRef[]>([])
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [uploadingImage, setUploadingImage] = useState(false)
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

  // Image upload is its own request against the existing post, independent
  // of Save, since /uploadPostImage is multipart and takes effect immediately.
  async function handleUploadImage() {
    if (!imageFile) return
    setUploadingImage(true)
    try {
      const imgUrl = await uploadPostImage(post.PostId, imageFile)
      showMessage('Image updated.', 'success')
      onImageUploaded(imgUrl)
      setImageFile(null)
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setUploadingImage(false)
    }
  }

  return (
    <div id="edit-post-form">
      <input type="text" value={title} maxLength={100} required onChange={(event) => setTitle(event.target.value)} />
      <textarea value={content} maxLength={2000} rows={4} cols={50} required onChange={(event) => setContent(event.target.value)} />
      <CategoryPicker selected={categories} onChange={setCategories} />
      <div className="edit-post-image">
        <input
          type="file"
          aria-label="Image"
          accept="image/png, image/jpeg, image/gif, image/webp"
          onChange={(event) => setImageFile(event.target.files?.[0] ?? null)}
        />
        <LoadingButton
          type="button"
          className="btns"
          loading={uploadingImage}
          loadingText="Uploading..."
          disabled={!imageFile}
          onClick={() => void handleUploadImage()}
        >
          {post.ImgURL ? 'Replace Image' : 'Upload Image'}
        </LoadingButton>
      </div>
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

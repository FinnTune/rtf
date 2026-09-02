import { useState } from 'react'
import { deleteCategory, editCategory } from '../../api/categories'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Category } from '../../types'
import { LoadingButton } from '../common/LoadingButton'

interface ManageCategoryRowProps {
  category: Category
  onChanged: () => void
}

export function ManageCategoryRow({ category, onChanged }: ManageCategoryRowProps) {
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(category.name)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const { showMessage } = useStatusMessage()

  async function handleSave() {
    const trimmed = name.trim()
    if (!trimmed) return
    setSaving(true)
    try {
      await editCategory(category.id, trimmed)
      showMessage('Category renamed.', 'success')
      setEditing(false)
      onChanged()
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      setSaving(false)
    }
  }

  async function handleDelete() {
    if (!window.confirm(`Delete the category "${category.name}"? Posts keep their other categories, if any.`)) {
      return
    }
    setDeleting(true)
    try {
      await deleteCategory(category.id)
      showMessage('Category deleted.', 'success')
      onChanged()
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      setDeleting(false)
    }
  }

  if (editing) {
    return (
      <li className="manage-category-row">
        <input type="text" aria-label="Category name" value={name} maxLength={30} onChange={(event) => setName(event.target.value)} />
        <LoadingButton type="button" className="btns btn-primary" loading={saving} loadingText="Saving..." onClick={() => void handleSave()}>
          Save
        </LoadingButton>
        <button
          type="button"
          className="btns"
          onClick={() => {
            setEditing(false)
            setName(category.name)
          }}
        >
          Cancel
        </button>
      </li>
    )
  }

  return (
    <li className="manage-category-row">
      <span>{category.name}</span>
      <button type="button" className="btns" onClick={() => setEditing(true)}>
        Rename
      </button>
      <LoadingButton type="button" className="btns btn-danger" loading={deleting} loadingText="Deleting..." onClick={() => void handleDelete()}>
        Delete
      </LoadingButton>
    </li>
  )
}

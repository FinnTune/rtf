import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { createCategory, getCategories } from '../../api/categories'
import { useAuth } from '../../contexts/AuthContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Category } from '../../types'
import { LoadingButton } from '../common/LoadingButton'
import { ManageCategoryRow } from './ManageCategoryRow'

export function ManageCategoriesPage() {
  const { user } = useAuth()

  // The backend is the real gate (RequireAdmin re-verifies on every write
  // regardless of what the client claims) — this just avoids showing a
  // page whose every action would visibly fail for a non-admin who
  // navigates here directly.
  if (user?.role !== 'admin') {
    return <p className="empty-state">Admin access required.</p>
  }

  return <ManageCategoriesForm />
}

function ManageCategoriesForm() {
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)
  const { showMessage } = useStatusMessage()

  const load = useCallback(() => {
    setLoading(true)
    getCategories()
      .then(setCategories)
      .catch((error: unknown) => {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
      .finally(() => setLoading(false))
  }, [showMessage])

  useEffect(() => {
    load()
  }, [load])

  async function handleCreate(event: FormEvent) {
    event.preventDefault()
    const trimmed = newName.trim()
    if (!trimmed) return
    setCreating(true)
    try {
      await createCategory(trimmed)
      setNewName('')
      showMessage('Category created.', 'success')
      load()
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setCreating(false)
    }
  }

  return (
    <div id="manage-categories">
      <h3>Manage Categories</h3>
      <form onSubmit={handleCreate}>
        <input
          type="text"
          value={newName}
          maxLength={30}
          placeholder="New category name"
          aria-label="New category name"
          onChange={(event) => setNewName(event.target.value)}
        />
        <LoadingButton type="submit" className="btns btn-primary" loading={creating} loadingText="Adding...">
          Add Category
        </LoadingButton>
      </form>
      {!loading && categories.length === 0 && <p className="empty-state">No categories yet.</p>}
      <ul className="manage-category-list">
        {categories.map((category) => (
          <ManageCategoryRow key={category.id} category={category} onChanged={load} />
        ))}
      </ul>
    </div>
  )
}

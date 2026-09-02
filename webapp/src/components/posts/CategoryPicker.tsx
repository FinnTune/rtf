import { useCallback, useEffect, useState } from 'react'
import { getCategories, subscribeToCategoryChanges } from '../../api/categories'
import type { PostCategoryRef } from '../../api/posts'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Category } from '../../types'

interface CategoryPickerProps {
  selected: PostCategoryRef[]
  onChange: (selected: PostCategoryRef[]) => void
}

// Shared by AddPostForm and PostEditForm. The original app actually had two
// different UIs for this (a collapsible dropdown for adding, inline pill
// checkboxes for editing) — unified here into one, using the dropdown
// styling for both, since maintaining two patterns for the same concept
// wasn't buying anything.
export function CategoryPicker({ selected, onChange }: CategoryPickerProps) {
  const [categories, setCategories] = useState<Category[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const { showMessage } = useStatusMessage()

  const load = useCallback(() => {
    getCategories()
      .then(setCategories)
      .catch((error: unknown) => {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
  }, [showMessage])

  useEffect(() => {
    load()
    return subscribeToCategoryChanges(load)
  }, [load])

  const selectedIds = new Set(selected.map((category) => category.id))

  function toggle(category: Category) {
    if (selectedIds.has(category.id)) {
      onChange(selected.filter((c) => c.id !== category.id))
    } else {
      onChange([...selected, { id: category.id, name: category.name }])
    }
  }

  return (
    <div className="dropdown">
      <button type="button" className="dropdown-toggle" aria-expanded={isOpen} onClick={() => setIsOpen((open) => !open)}>
        Select Categories&gt;&gt;
      </button>
      {isOpen && (
        <div className="dropdown-content">
          {categories.map((category) => (
            <label key={category.id}>
              <input type="checkbox" checked={selectedIds.has(category.id)} onChange={() => toggle(category)} />
              {category.name}
            </label>
          ))}
        </div>
      )}
    </div>
  )
}

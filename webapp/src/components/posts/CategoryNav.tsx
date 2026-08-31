import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getCategories, subscribeToCategoryChanges } from '../../api/categories'
import { useFeedView } from '../../contexts/FeedViewContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Category } from '../../types'

export function CategoryNav() {
  const [categories, setCategories] = useState<Category[]>([])
  const { view, toggleCategory } = useFeedView()
  const { showMessage } = useStatusMessage()
  const navigate = useNavigate()

  const load = useCallback(() => {
    getCategories()
      .then(setCategories)
      .catch((error: unknown) => {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
  }, [showMessage])

  useEffect(() => {
    load()
    // This component stays mounted for the whole logged-in session (it's
    // part of the persistent sidebar, not remounted on navigation), so it
    // needs to hear about admin edits made elsewhere, not just fetch once.
    return subscribeToCategoryChanges(load)
  }, [load])

  const selectedNames = view.type === 'category' ? view.names : []

  return (
    <>
      {categories.map((category) => {
        const isActive = selectedNames.includes(category.name)
        return (
          <button
            key={category.id}
            type="button"
            aria-pressed={isActive}
            className={isActive ? 'nav-item category-nav-item active' : 'nav-item category-nav-item'}
            onClick={() => {
              toggleCategory(category.name)
              navigate('/')
            }}
          >
            {category.name}
          </button>
        )
      })}
    </>
  )
}

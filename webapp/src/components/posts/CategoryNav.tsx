import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getCategories } from '../../api/categories'
import { useFeedView } from '../../contexts/FeedViewContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { Category } from '../../types'

export function CategoryNav() {
  const [categories, setCategories] = useState<Category[]>([])
  const { view, showCategory } = useFeedView()
  const { showMessage } = useStatusMessage()
  const navigate = useNavigate()

  useEffect(() => {
    getCategories()
      .then(setCategories)
      .catch((error: unknown) => {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
  }, [showMessage])

  return (
    <>
      {categories.map((category) => {
        const isActive = view.type === 'category' && view.name === category.name
        return (
          <button
            key={category.id}
            type="button"
            className={isActive ? 'nav-item category-nav-item active' : 'nav-item category-nav-item'}
            onClick={() => {
              showCategory(category.name)
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

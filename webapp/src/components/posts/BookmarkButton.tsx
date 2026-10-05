import { useBookmark } from '../../hooks/useBookmark'
import type { Post } from '../../types'

export function BookmarkButton({ post }: { post: Post }) {
  const { bookmarked, pending, toggle } = useBookmark(post)

  return (
    <button
      type="button"
      className={bookmarked ? 'btns reaction-btn active' : 'btns reaction-btn'}
      disabled={pending}
      aria-pressed={bookmarked}
      onClick={(event) => {
        event.preventDefault()
        toggle()
      }}
    >
      {bookmarked ? 'Bookmarked' : 'Bookmark'}
    </button>
  )
}

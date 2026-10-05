import { useState } from 'react'
import { bookmarkPost } from '../api/posts'
import { useStatusMessage } from '../contexts/StatusMessageContext'
import type { Post } from '../types'

// Simpler than useReaction: a bookmark is a plain yes/no with no
// aggregate count or live broadcast to reconcile against, no switch-
// between-two-values case — just an optimistic toggle, rolled back on
// error, same as every other optimistic-update hook in this app.
export function useBookmark(post: Post) {
  const [bookmarked, setBookmarked] = useState(post.MyBookmark)
  const [pending, setPending] = useState(false)
  const { showMessage } = useStatusMessage()

  async function toggle() {
    if (pending) return
    const previous = bookmarked
    setBookmarked(!previous)
    setPending(true)
    try {
      const result = await bookmarkPost(post.PostId)
      setBookmarked(result)
    } catch (error) {
      setBookmarked(previous)
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setPending(false)
    }
  }

  return { bookmarked, pending, toggle: () => void toggle() }
}

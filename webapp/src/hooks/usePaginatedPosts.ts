import { useCallback, useEffect, useRef, useState, type DependencyList } from 'react'
import { useStatusMessage } from '../contexts/StatusMessageContext'
import type { Post } from '../types'

interface PageResult {
  posts: Post[]
  total: number | null
}

type Fetcher = (offset: number, limit: number) => Promise<PageResult>

// Shared logic behind every paginated post list (all posts, one category,
// one author, search) — offset math, loading state, and reporting fetch
// errors through the global status banner, matching every one of the
// original app's near-identical fetch+render call sites.
//
// `deps` re-runs the fetch from page 1 whenever the thing being browsed
// changes (a different category, author, or search query). It's forwarded
// straight to useEffect, so callers get real exhaustive-deps checking at
// their own call site — this hook can't statically declare deps for them.
export function usePaginatedPosts(fetcher: Fetcher, deps: DependencyList, pageSize = 10) {
  const { showMessage } = useStatusMessage()
  const [offset, setOffset] = useState(0)
  const [posts, setPosts] = useState<Post[]>([])
  const [total, setTotal] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)

  // The empty-page snap-back below needs to call the latest `load` again,
  // but referencing `load` from inside its own useCallback factory isn't
  // safe to statically verify — go through a ref instead.
  const loadRef = useRef<(targetOffset: number) => void>(() => {})

  const load = useCallback(
    (targetOffset: number) => {
      setLoading(true)
      fetcher(targetOffset, pageSize)
        .then((result) => {
          if (targetOffset > 0 && result.posts.length === 0) {
            // The page we asked for is now empty (e.g. its last post was
            // deleted elsewhere) — snap back a page instead of a dead end.
            loadRef.current(Math.max(0, targetOffset - pageSize))
            return
          }
          setPosts(result.posts)
          setTotal(result.total)
          setOffset(targetOffset)
        })
        .catch((error: unknown) => {
          showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
        })
        .finally(() => setLoading(false))
    },
    [fetcher, pageSize, showMessage],
  )

  useEffect(() => {
    loadRef.current = load
  }, [load])

  useEffect(() => {
    load(0)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- deps is the caller's own dependency list, forwarded as-is
  }, deps)

  return { posts, total, offset, pageSize, loading, goToOffset: load }
}

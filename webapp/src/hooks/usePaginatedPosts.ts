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

  // Nothing here cancels the underlying fetch or guarantees responses land
  // in request order — a category/sort/search change (deps) or a fast
  // double pagination click can have two requests in flight at once, and
  // whichever HTTP response arrives last wins regardless of which was
  // issued last. requestIdRef lets each call recognize when it's been
  // superseded so a stale response can't overwrite newer state (and, as a
  // side effect, so a superseded call's own `finally` can't flash `loading`
  // back to false while the snap-back re-fetch below is still in flight).
  const requestIdRef = useRef(0)

  const load = useCallback(
    (targetOffset: number) => {
      const requestId = ++requestIdRef.current
      setLoading(true)
      fetcher(targetOffset, pageSize)
        .then((result) => {
          if (requestId !== requestIdRef.current) return
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
          if (requestId !== requestIdRef.current) return
          showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
        })
        .finally(() => {
          if (requestId !== requestIdRef.current) return
          setLoading(false)
        })
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

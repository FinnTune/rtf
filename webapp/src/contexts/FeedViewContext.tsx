import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'

// What the feed at "/" is currently browsing. The sidebar category nav and
// the topbar search form both need to drive this, but they're DOM siblings
// of the feed (not parent/child) in the app shell — hence a context rather
// than component state or prop drilling.
export type FeedView = { type: 'all' } | { type: 'category'; name: string } | { type: 'search'; query: string }

interface FeedViewContextValue {
  view: FeedView
  showAllPosts: () => void
  showCategory: (name: string) => void
  showSearch: (query: string) => void
  // Bumped by showAllPosts/showCategory only (not showSearch) so the topbar
  // search box can key off it and reset itself — via remounting, not a
  // setState-in-effect sync — whenever the view moves away from search for
  // a reason other than the search box's own submit. See Topbar.tsx.
  searchResetToken: number
}

const FeedViewContext = createContext<FeedViewContextValue | null>(null)

export function FeedViewProvider({ children }: { children: ReactNode }) {
  const [view, setView] = useState<FeedView>({ type: 'all' })
  const [searchResetToken, setSearchResetToken] = useState(0)

  const showAllPosts = useCallback(() => {
    setView({ type: 'all' })
    setSearchResetToken((token) => token + 1)
  }, [])
  const showCategory = useCallback((name: string) => {
    setView({ type: 'category', name })
    setSearchResetToken((token) => token + 1)
  }, [])
  const showSearch = useCallback((query: string) => setView({ type: 'search', query }), [])

  return (
    <FeedViewContext.Provider value={{ view, showAllPosts, showCategory, showSearch, searchResetToken }}>
      {children}
    </FeedViewContext.Provider>
  )
}

export function useFeedView(): FeedViewContextValue {
  const ctx = useContext(FeedViewContext)
  if (!ctx) {
    throw new Error('useFeedView must be used within a FeedViewProvider')
  }
  return ctx
}

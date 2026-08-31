import { useCallback } from 'react'
import { getAllPosts, getPostsByCategory, searchPosts } from '../../api/posts'
import { useFeedView } from '../../contexts/FeedViewContext'
import { usePaginatedPosts } from '../../hooks/usePaginatedPosts'
import { PostList } from './PostList'

// The "/" route: all posts, a category filter, or search results,
// depending on FeedViewContext — set from the sidebar category nav or the
// topbar search form, both siblings of this component in the shell.
export function Feed() {
  const { view } = useFeedView()

  const fetcher = useCallback(
    (offset: number, limit: number) => {
      if (view.type === 'category') return getPostsByCategory(view.names, offset, limit)
      if (view.type === 'search') return searchPosts(view.query)
      return getAllPosts(offset, limit)
    },
    [view],
  )

  const emptyMessage =
    view.type === 'category'
      ? 'No posts for this category.'
      : view.type === 'search'
        ? `No posts found for "${view.query}".`
        : 'No posts yet — be the first to post!'
  // Note: the "this category" phrasing above stays singular even with
  // multiple selected, matching how it read before multi-select — a
  // deliberate, minor choice not to over-specify wording for an edge case.

  const { posts, total, offset, pageSize, loading, goToOffset } = usePaginatedPosts(fetcher, [view], 10)

  return (
    <PostList
      posts={posts}
      total={total}
      offset={offset}
      pageSize={pageSize}
      loading={loading}
      emptyMessage={emptyMessage}
      onNavigate={goToOffset}
    />
  )
}

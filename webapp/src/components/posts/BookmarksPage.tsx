import { getBookmarks } from '../../api/posts'
import { usePaginatedPosts } from '../../hooks/usePaginatedPosts'
import { PostList } from './PostList'

export function BookmarksPage() {
  const { posts, total, offset, pageSize, loading, goToOffset } = usePaginatedPosts(getBookmarks, [], 10)

  return (
    <PostList
      posts={posts}
      total={total}
      offset={offset}
      pageSize={pageSize}
      loading={loading}
      heading="Bookmarked Posts"
      emptyMessage="You haven't bookmarked any posts yet."
      onNavigate={goToOffset}
    />
  )
}

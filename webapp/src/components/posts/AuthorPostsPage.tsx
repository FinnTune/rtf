import { useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { getPostsByAuthor } from '../../api/posts'
import { usePaginatedPosts } from '../../hooks/usePaginatedPosts'
import { PostList } from './PostList'

export function AuthorPostsPage() {
  const { username = '' } = useParams()

  const fetcher = useCallback((offset: number, limit: number) => getPostsByAuthor(username, offset, limit), [username])
  const { posts, total, offset, pageSize, loading, goToOffset } = usePaginatedPosts(fetcher, [username], 10)

  return (
    <PostList
      posts={posts}
      total={total}
      offset={offset}
      pageSize={pageSize}
      loading={loading}
      heading={`Posts by ${username}`}
      emptyMessage={`${username} hasn't posted yet.`}
      onNavigate={goToOffset}
    />
  )
}

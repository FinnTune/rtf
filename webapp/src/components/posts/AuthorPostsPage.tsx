import { useCallback, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getPostsByAuthor, type PostSort } from '../../api/posts'
import { usePaginatedPosts } from '../../hooks/usePaginatedPosts'
import { PostList } from './PostList'
import { PostSortSelect } from './PostSortSelect'

export function AuthorPostsPage() {
  const { username = '' } = useParams()
  const [sort, setSort] = useState<PostSort>('newest')

  const fetcher = useCallback(
    (offset: number, limit: number) => getPostsByAuthor(username, offset, limit, sort),
    [username, sort],
  )
  const { posts, total, offset, pageSize, loading, goToOffset } = usePaginatedPosts(fetcher, [username, sort], 10)

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
      sortControl={<PostSortSelect value={sort} onChange={setSort} />}
    />
  )
}

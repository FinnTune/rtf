import type { ReactNode } from 'react'
import type { Post } from '../../types'
import { Pagination } from './Pagination'
import { PostCard } from './PostCard'

interface PostListProps {
  posts: Post[]
  total: number | null
  offset: number
  pageSize: number
  loading: boolean
  heading?: string
  emptyMessage: string
  onNavigate: (offset: number) => void
  sortControl?: ReactNode
}

// Shared by the all-posts feed, category filter, search results, and
// author-posts views — replaces the original app's four near-identical
// copies of createPostsTable/renderPostRows/renderPagination/showEmptyState.
export function PostList({
  posts,
  total,
  offset,
  pageSize,
  loading,
  heading = 'Latest Posts',
  emptyMessage,
  onNavigate,
  sortControl,
}: PostListProps) {
  const showEmptyState = posts.length === 0 && !loading

  return (
    <div id="posts">
      <div className="post-list-header">
        <h3>{heading}</h3>
        {sortControl}
      </div>
      <ul className="post-list" id="posts-table">
        {showEmptyState ? (
          <li className="empty-state">{emptyMessage}</li>
        ) : (
          posts.map((post) => <PostCard key={post.PostId} post={post} />)
        )}
      </ul>
      {total !== null && total > 0 && (
        <Pagination offset={offset} pageSize={pageSize} total={total} loading={loading} onNavigate={onNavigate} />
      )}
    </div>
  )
}

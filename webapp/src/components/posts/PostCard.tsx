import { Link } from 'react-router-dom'
import type { Post } from '../../types'

export function PostCard({ post }: { post: Post }) {
  return (
    <li className="post-card">
      <div className="post-card-title">
        <Link to={`/posts/${post.PostId}`}>{post.Title}</Link>
      </div>
      <div className="post-card-preview">{post.Content}</div>
      <div className="post-card-meta">
        <Link to={`/users/${encodeURIComponent(post.Author)}`} className="author-link">
          {post.Author}
        </Link>
        {' · '}
        {post.Created}
      </div>
    </li>
  )
}

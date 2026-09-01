import { Link } from 'react-router-dom'
import type { Post } from '../../types'
import { ReactionButtons } from './ReactionButtons'

export function PostCard({ post }: { post: Post }) {
  return (
    <li className="post-card">
      <div className="post-card-title">
        <Link to={`/posts/${post.PostId}`}>{post.Title}</Link>
      </div>
      {post.ImgURL && (
        <Link to={`/posts/${post.PostId}`}>
          <img src={post.ImgURL} alt={post.Title} className="post-card-image" />
        </Link>
      )}
      <div className="post-card-preview">{post.Content}</div>
      <div className="post-card-meta">
        <Link to={`/users/${encodeURIComponent(post.Author)}`} className="author-link">
          {post.Author}
        </Link>
        {' · '}
        {post.Created}
      </div>
      <ReactionButtons post={post} />
    </li>
  )
}

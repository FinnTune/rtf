import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { getUserProfile, type UserProfile } from '../../api/profile'
import { getPostsByAuthor, type PostSort } from '../../api/posts'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import { usePaginatedPosts } from '../../hooks/usePaginatedPosts'
import { PostList } from './PostList'
import { PostSortSelect } from './PostSortSelect'

export function AuthorPostsPage() {
  const { username = '' } = useParams()
  const [sort, setSort] = useState<PostSort>('newest')
  const { showMessage } = useStatusMessage()

  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [notFound, setNotFound] = useState(false)

  // Independent of the post list's own fetch below — GetPostsByAuthorHandler
  // doesn't check the username actually exists (post.author is a loose
  // string reference, not a FK — see UpdateProfileHandler's doc comment),
  // so a nonexistent username's post list comes back merely empty, not an
  // error. This profile fetch is what actually 404s, which is what decides
  // whether this page shows "user not found" at all.
  useEffect(() => {
    setProfile(null)
    setNotFound(false)
    getUserProfile(username)
      .then(setProfile)
      .catch((error: unknown) => {
        setNotFound(true)
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
  }, [username, showMessage])

  const fetcher = useCallback(
    (offset: number, limit: number) => getPostsByAuthor(username, offset, limit, sort),
    [username, sort],
  )
  const { posts, total, offset, pageSize, loading, goToOffset } = usePaginatedPosts(fetcher, [username, sort], 10)

  if (notFound) {
    return <p className="empty-state">User not found.</p>
  }

  return (
    <div id="author-profile">
      {profile && (
        <div className="profile-summary">
          <h3>{profile.username}</h3>
          <p>
            Joined {profile.joined} · {profile.post_count} {profile.post_count === 1 ? 'post' : 'posts'} ·{' '}
            {profile.comment_count} {profile.comment_count === 1 ? 'comment' : 'comments'}
          </p>
        </div>
      )}
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
    </div>
  )
}

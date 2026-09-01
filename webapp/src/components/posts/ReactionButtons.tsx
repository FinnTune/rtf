import { useReaction } from '../../hooks/useReaction'
import type { Post } from '../../types'

export function ReactionButtons({ post }: { post: Post }) {
  const { likeCount, dislikeCount, myReaction, pending, like, dislike } = useReaction(post)

  return (
    <div className="reaction-buttons">
      <button
        type="button"
        className={myReaction === 'liked' ? 'btns reaction-btn active' : 'btns reaction-btn'}
        disabled={pending}
        onClick={(event) => {
          event.preventDefault()
          like()
        }}
      >
        Like ({likeCount})
      </button>
      <button
        type="button"
        className={myReaction === 'disliked' ? 'btns reaction-btn active' : 'btns reaction-btn'}
        disabled={pending}
        onClick={(event) => {
          event.preventDefault()
          dislike()
        }}
      >
        Dislike ({dislikeCount})
      </button>
    </div>
  )
}

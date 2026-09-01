import { useState } from 'react'
import { reactToPost } from '../api/posts'
import { useStatusMessage } from '../contexts/StatusMessageContext'
import type { Post } from '../types'

interface ReactionState {
  likeCount: number
  dislikeCount: number
  myReaction: string
}

// Mirrors ReactToPostHandler's own state machine (submitting the same
// reaction again toggles it off; submitting the opposite one switches it)
// so the UI updates instantly on click, then reconciles with the server's
// authoritative response once it arrives — correcting for drift if e.g.
// another user reacted to the same post in between.
function applyOptimistic(state: ReactionState, wantLiked: boolean): ReactionState {
  const wasLiked = state.myReaction === 'liked'
  const wasDisliked = state.myReaction === 'disliked'
  const isTogglingOff = (wantLiked && wasLiked) || (!wantLiked && wasDisliked)

  let { likeCount, dislikeCount } = state
  if (wasLiked) likeCount -= 1
  if (wasDisliked) dislikeCount -= 1

  if (isTogglingOff) {
    return { likeCount, dislikeCount, myReaction: 'none' }
  }
  if (wantLiked) likeCount += 1
  else dislikeCount += 1
  return { likeCount, dislikeCount, myReaction: wantLiked ? 'liked' : 'disliked' }
}

export function useReaction(post: Post) {
  const [state, setState] = useState<ReactionState>({
    likeCount: post.LikeCount,
    dislikeCount: post.DislikeCount,
    myReaction: post.MyReaction,
  })
  const [pending, setPending] = useState(false)
  const { showMessage } = useStatusMessage()

  async function react(wantLiked: boolean) {
    if (pending) return
    const previous = state
    setState(applyOptimistic(state, wantLiked))
    setPending(true)
    try {
      const result = await reactToPost(post.PostId, wantLiked)
      setState({ likeCount: result.likeCount, dislikeCount: result.dislikeCount, myReaction: result.myReaction })
    } catch (error) {
      setState(previous)
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setPending(false)
    }
  }

  return {
    likeCount: state.likeCount,
    dislikeCount: state.dislikeCount,
    myReaction: state.myReaction,
    pending,
    like: () => void react(true),
    dislike: () => void react(false),
  }
}

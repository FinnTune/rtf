import { useEffect, useState } from 'react'
import { reactToPost } from '../api/posts'
import { useStatusMessage } from '../contexts/StatusMessageContext'
import { useOptionalWebSocket } from '../contexts/WebSocketContext'
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
  const ws = useOptionalWebSocket()

  // Keeps this post's counts live for every OTHER connected client's own
  // reactions too — the server broadcasts post-reaction-updated (excluding
  // the reacting client itself, which already gets the authoritative
  // counts via reactToPost's own response below) whenever anyone reacts.
  // Never touches myReaction: that's personal, and the broadcast carries
  // no such field (see PostReactionUpdatedEvent's doc comment on the Go
  // side) — only this viewer's own react() call ever changes it.
  useEffect(() => {
    if (!ws) return
    return ws.subscribe('post-reaction-updated', (payload) => {
      const update = payload as { post_id: number; like_count: number; dislike_count: number }
      if (update.post_id !== post.PostId) return
      setState((prev) => ({ ...prev, likeCount: update.like_count, dislikeCount: update.dislike_count }))
    })
  }, [ws, post.PostId])

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

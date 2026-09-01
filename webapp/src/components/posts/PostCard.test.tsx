import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import type { Post } from '../../types'
import { PostCard } from './PostCard'

function makePost(overrides: Partial<Post> = {}): Post {
  return {
    PostId: 1,
    UserId: 1,
    Title: 'Test Post',
    Content: 'Body',
    Author: 'alice',
    Created: '2026-01-01',
    ImgURL: '',
    LikeCount: 0,
    DislikeCount: 0,
    MyReaction: 'none',
    ...overrides,
  }
}

function renderCard(post: Post) {
  return render(
    <MemoryRouter>
      <StatusMessageProvider>
        <ul>
          <PostCard post={post} />
        </ul>
      </StatusMessageProvider>
    </MemoryRouter>,
  )
}

describe('PostCard', () => {
  it('renders an image linking to the post when ImgURL is set', () => {
    renderCard(makePost({ ImgURL: '/uploads/posts/abc.png' }))
    const img = screen.getByRole('img', { name: 'Test Post' })
    expect(img).toHaveAttribute('src', '/uploads/posts/abc.png')
  })

  it('renders no image when ImgURL is empty', () => {
    renderCard(makePost())
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })
})

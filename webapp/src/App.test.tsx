import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import App from './App'

describe('App routing', () => {
  it('renders the feed placeholder at /', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>,
    )
    expect(screen.getByText(/Feed — coming soon/)).toBeInTheDocument()
  })

  it('renders the single-post placeholder at /posts/:id', () => {
    render(
      <MemoryRouter initialEntries={['/posts/42']}>
        <App />
      </MemoryRouter>,
    )
    expect(screen.getByText(/Post — coming soon/)).toBeInTheDocument()
  })
})

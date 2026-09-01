import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { FeedViewProvider } from '../../contexts/FeedViewContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { Feed } from './Feed'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function renderFeed() {
  return render(
    <StatusMessageProvider>
      <FeedViewProvider>
        <Feed />
      </FeedViewProvider>
    </StatusMessageProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Feed sorting', () => {
  it('fetches with sort=newest by default and re-fetches with the new sort on change', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/getAllPosts')) {
        return new Response(JSON.stringify([]), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      throw new Error('Unexpected fetch in test: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    renderFeed()

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/getAllPosts?limit=10&offset=0&sort=newest'), undefined),
    )

    await userEvent.selectOptions(screen.getByLabelText('Sort posts by'), 'most_liked')

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/getAllPosts?limit=10&offset=0&sort=most_liked'), undefined),
    )
  })
})

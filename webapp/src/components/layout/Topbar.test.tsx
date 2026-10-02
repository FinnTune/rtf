import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { FeedViewProvider, useFeedView } from '../../contexts/FeedViewContext'
import { checkLoginResponse } from '../../testUtils/chatTestHarness'
import { Topbar } from './Topbar'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

// Surfaces the current feed view next to Topbar so tests can assert on it
// directly, the same way Feed.tsx would consume it — Topbar itself never
// renders view.type anywhere.
function ViewProbe() {
  const { view } = useFeedView()
  return <p data-testid="view-probe">{view.type}</p>
}

function mockBackend() {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) {
        return checkLoginResponse('alice')
      }
      if (url.startsWith('/logout')) {
        return new Response(null, { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    }),
  )
}

function renderTopbar() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <AuthProvider>
        <FeedViewProvider>
          <Topbar />
          <ViewProbe />
        </FeedViewProvider>
      </AuthProvider>
    </MemoryRouter>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Topbar', () => {
  it("shows the logged-in user's username", async () => {
    mockBackend()
    renderTopbar()
    expect(await screen.findByText('alice')).toHaveAttribute('id', 'topbar-username')
  })

  it('submitting a search moves the feed view to search and shows Clear', async () => {
    mockBackend()
    renderTopbar()
    await screen.findByText('alice')

    expect(screen.queryByRole('button', { name: 'Clear' })).not.toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Search posts'), 'dragons')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => expect(screen.getByTestId('view-probe')).toHaveTextContent('search'))
    expect(screen.getByRole('button', { name: 'Clear' })).toBeInTheDocument()
  })

  it('Clear returns the view to all and remounts SearchBox, discarding any unsubmitted typed text', async () => {
    mockBackend()
    renderTopbar()
    await screen.findByText('alice')

    await userEvent.type(screen.getByLabelText('Search posts'), 'dragons')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))
    await waitFor(() => expect(screen.getByTestId('view-probe')).toHaveTextContent('search'))

    // Typed but never submitted — a plain setState-synced input would keep
    // showing this after Clear; the key-remount is what actually drops it.
    await userEvent.type(screen.getByLabelText('Search posts'), 'unsubmitted text')

    await userEvent.click(screen.getByRole('button', { name: 'Clear' }))
    await waitFor(() => expect(screen.getByTestId('view-probe')).toHaveTextContent('all'))
    expect(screen.queryByRole('button', { name: 'Clear' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Search posts')).toHaveValue('')
  })

  it('logging out clears the displayed username', async () => {
    mockBackend()
    renderTopbar()
    await screen.findByText('alice')

    await userEvent.click(screen.getByRole('button', { name: 'Logout' }))
    await waitFor(() => expect(screen.queryByText('alice')).not.toBeInTheDocument())

    const fetchMock = vi.mocked(fetch)
    expect(fetchMock.mock.calls.some(([input]) => requestUrl(input as string | URL | Request).startsWith('/logout'))).toBe(true)
  })
})

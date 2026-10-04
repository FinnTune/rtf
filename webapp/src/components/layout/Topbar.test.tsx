import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { FeedViewProvider, useFeedView } from '../../contexts/FeedViewContext'
import { NotificationsProvider } from '../../contexts/NotificationsContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { WebSocketProvider } from '../../contexts/WebSocketContext'
import { ControllableFakeWebSocket, checkLoginResponse } from '../../testUtils/chatTestHarness'
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
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
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
      if (url.startsWith('/notifications')) {
        return new Response(JSON.stringify({ notifications: [], unread_count: 0 }), { status: 200, headers: { 'X-Total-Count': '0' } })
      }
      throw new Error('Unexpected fetch: ' + url)
    }),
  )
}

async function renderTopbar() {
  const result = render(
    <MemoryRouter initialEntries={['/']}>
      <StatusMessageProvider>
        <AuthProvider>
          <WebSocketProvider>
            <NotificationsProvider>
              <FeedViewProvider>
                <Topbar />
                <ViewProbe />
              </FeedViewProvider>
            </NotificationsProvider>
          </WebSocketProvider>
        </AuthProvider>
      </StatusMessageProvider>
    </MemoryRouter>,
  )
  // Settles the WS connection before returning — see NotificationsBell.test.tsx's
  // identical comment on why every WS-dependent test does this.
  await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  act(() => ControllableFakeWebSocket.instances[0].simulateOpen())
  return result
}

afterEach(() => {
  vi.unstubAllGlobals()
  ControllableFakeWebSocket.instances = []
})

describe('Topbar', () => {
  it("shows the logged-in user's username", async () => {
    mockBackend()
    await renderTopbar()
    expect(await screen.findByText('alice')).toHaveAttribute('id', 'topbar-username')
  })

  it('submitting a search moves the feed view to search and shows Clear', async () => {
    mockBackend()
    await renderTopbar()
    await screen.findByText('alice')

    expect(screen.queryByRole('button', { name: 'Clear' })).not.toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Search posts'), 'dragons')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => expect(screen.getByTestId('view-probe')).toHaveTextContent('search'))
    expect(screen.getByRole('button', { name: 'Clear' })).toBeInTheDocument()
  })

  it('Clear returns the view to all and remounts SearchBox, discarding any unsubmitted typed text', async () => {
    mockBackend()
    await renderTopbar()
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
    await renderTopbar()
    await screen.findByText('alice')

    await userEvent.click(screen.getByRole('button', { name: 'Logout' }))
    await waitFor(() => expect(screen.queryByText('alice')).not.toBeInTheDocument())

    const fetchMock = vi.mocked(fetch)
    expect(fetchMock.mock.calls.some(([input]) => requestUrl(input as string | URL | Request).startsWith('/logout'))).toBe(true)
  })
})

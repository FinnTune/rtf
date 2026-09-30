import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { StatusBanner } from '../common/StatusBanner'
import { ManageUsersPage } from './ManageUsersPage'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function checkLoginResponse(role: string) {
  return new Response(
    JSON.stringify({ loggedIn: true, id: 1, username: 'admin', email: 'admin@example.com', joined: '2026-01-01', otp: 'x', role }),
    { status: 200 },
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ManageUsersPage', () => {
  it('shows "Admin access required" for a non-admin user, without ever fetching users', async () => {
    const fetchMock = vi.fn().mockResolvedValue(checkLoginResponse('user'))
    vi.stubGlobal('fetch', fetchMock)
    render(
      <StatusMessageProvider>
        <AuthProvider>
          <ManageUsersPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )
    expect(await screen.findByText('Admin access required.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([input]) => requestUrl(input as string | URL | Request).startsWith('/listUsers'))).toBe(false)
  })

  it('lets an admin view and ban/unban a user, but never ban themselves', async () => {
    let users = [
      { id: 1, username: 'admin', email: 'admin@example.com', role: 'admin', banned: false },
      { id: 2, username: 'bob', email: 'bob@example.com', role: 'user', banned: false },
    ]

    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) {
        return checkLoginResponse('admin')
      }
      if (url.startsWith('/listUsers')) {
        return new Response(JSON.stringify(users), { status: 200 })
      }
      if (url.startsWith('/setUserBanned')) {
        const { user_id, banned } = JSON.parse(init!.body as string)
        users = users.map((u) => (u.id === user_id ? { ...u, banned } : u))
        return new Response('', { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <StatusBanner />
          <ManageUsersPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    expect(await screen.findByText(/bob/)).toBeInTheDocument()

    // The admin's own row has a disabled Ban button, with a reason exposed
    // (not just silently disabled).
    const adminRow = screen.getByText(/admin@example.com/).closest('li')!
    const ownBanButton = within(adminRow).getByRole('button', { name: 'Ban' })
    expect(ownBanButton).toBeDisabled()
    expect(ownBanButton).toHaveAttribute('title', "You can't ban your own account")

    // Ban bob.
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const bobRow = screen.getByText(/bob@example.com/).closest('li')!
    await userEvent.click(within(bobRow).getByRole('button', { name: 'Ban' }))
    await waitFor(() => expect(within(bobRow).getByRole('button', { name: 'Unban' })).toBeInTheDocument())
    expect(within(bobRow).getByText(/— banned/)).toBeInTheDocument()

    // Unban bob.
    await userEvent.click(within(bobRow).getByRole('button', { name: 'Unban' }))
    await waitFor(() => expect(within(bobRow).getByRole('button', { name: 'Ban' })).toBeInTheDocument())
    expect(within(bobRow).queryByText(/— banned/)).not.toBeInTheDocument()
  })

  // Regression test: SetUserBannedHandler rejects banning a fellow admin
  // account server-side, but the button previously gave no indication of
  // that until the click actually failed — mirrors the existing self-ban
  // disabled-button treatment.
  it('disables the Ban button for a fellow admin, with a reason exposed, but allows unbanning one', async () => {
    let users = [
      { id: 1, username: 'admin', email: 'admin@example.com', role: 'admin', banned: false },
      { id: 2, username: 'otherAdmin', email: 'other-admin@example.com', role: 'admin', banned: true },
    ]

    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('admin')
      if (url.startsWith('/listUsers')) return new Response(JSON.stringify(users), { status: 200 })
      if (url.startsWith('/setUserBanned')) {
        const { user_id, banned } = JSON.parse(init!.body as string)
        users = users.map((u) => (u.id === user_id ? { ...u, banned } : u))
        return new Response('', { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <ManageUsersPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    const otherAdminRow = (await screen.findByText(/otherAdmin/)).closest('li')!
    // otherAdmin starts banned, so Unban must be enabled even though
    // they're a fellow admin — only the ban direction is blocked.
    const unbanButton = within(otherAdminRow).getByRole('button', { name: 'Unban' })
    expect(unbanButton).toBeEnabled()

    await userEvent.click(unbanButton)
    await waitFor(() => expect(within(otherAdminRow).getByRole('button', { name: 'Ban' })).toBeInTheDocument())

    const banButton = within(otherAdminRow).getByRole('button', { name: 'Ban' })
    expect(banButton).toBeDisabled()
    expect(banButton).toHaveAttribute('title', "You can't ban another admin account")
  })

  it('requests users with limit/offset and shows Previous/Next controls once there are more than one page', async () => {
    const page1 = Array.from({ length: 20 }, (_, i) => ({
      id: i + 1,
      username: `user${i + 1}`,
      email: `user${i + 1}@example.com`,
      role: 'user',
      banned: false,
    }))
    const page2 = [{ id: 21, username: 'user21', email: 'user21@example.com', role: 'user', banned: false }]

    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('admin')
      if (url.startsWith('/listUsers')) {
        const params = new URL(url, 'https://localhost').searchParams
        const offset = Number(params.get('offset'))
        const body = offset === 0 ? page1 : page2
        return new Response(JSON.stringify(body), { status: 200, headers: { 'X-Total-Count': '21' } })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <ManageUsersPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )
    await screen.findByText(/user1@/)

    // The very first request already carries a bounded limit/offset, not an
    // unbounded fetch-everything call.
    const firstListUsersCall = fetchMock.mock.calls.find(([input]) => requestUrl(input as string | URL | Request).startsWith('/listUsers'))
    expect(requestUrl(firstListUsersCall![0] as string | URL | Request)).toContain('limit=20')
    expect(requestUrl(firstListUsersCall![0] as string | URL | Request)).toContain('offset=0')

    const nextButton = screen.getByRole('button', { name: 'Next' })
    expect(screen.queryByText(/user21@/)).not.toBeInTheDocument()

    await userEvent.click(nextButton)
    await waitFor(() => expect(screen.getByText(/user21@/)).toBeInTheDocument())
    expect(screen.queryByText(/^user1@/)).not.toBeInTheDocument()
  })

  it('does not call the API when the ban confirmation is dismissed', async () => {
    const users = [{ id: 2, username: 'bob', email: 'bob@example.com', role: 'user', banned: false }]
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('admin')
      if (url.startsWith('/listUsers')) return new Response(JSON.stringify(users), { status: 200 })
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <ManageUsersPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )
    await screen.findByText(/bob/)

    vi.spyOn(window, 'confirm').mockReturnValue(false)
    await userEvent.click(screen.getByRole('button', { name: 'Ban' }))

    expect(fetchMock.mock.calls.some(([input]) => requestUrl(input as string | URL | Request).startsWith('/setUserBanned'))).toBe(
      false,
    )
  })

  // Regression test for the actual fix: ListUsersHandler's ?q= used to not
  // exist at all, so there was no way to search for a user besides paging
  // through the whole list alphabetically.
  it('submitting a search sends q= and resets back to offset 0', async () => {
    const allUsers = [
      { id: 1, username: 'admin', email: 'admin@example.com', role: 'admin', banned: false },
      { id: 2, username: 'alice', email: 'alice@example.com', role: 'user', banned: false },
    ]
    const filteredUsers = [allUsers[1]]

    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('admin')
      if (url.startsWith('/listUsers')) {
        const q = new URL(url, 'https://localhost').searchParams.get('q')
        const body = q ? filteredUsers : allUsers
        return new Response(JSON.stringify(body), { status: 200, headers: { 'X-Total-Count': String(body.length) } })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <ManageUsersPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )
    await screen.findByText(/admin@/)
    expect(screen.getByText(/alice@/)).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Search users by username or email'), 'alice')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => expect(screen.queryByText(/^admin@/)).not.toBeInTheDocument())
    expect(screen.getByText(/alice@/)).toBeInTheDocument()

    const searchCall = fetchMock.mock.calls.find(([input]) => requestUrl(input as string | URL | Request).includes('q=alice'))
    expect(searchCall).toBeDefined()
    expect(requestUrl(searchCall![0] as string | URL | Request)).toContain('offset=0')
  })

  it('a blank search still calls listUsers without a q param', async () => {
    const users = [{ id: 2, username: 'bob', email: 'bob@example.com', role: 'user', banned: false }]
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse('admin')
      if (url.startsWith('/listUsers')) return new Response(JSON.stringify(users), { status: 200 })
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <ManageUsersPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )
    await screen.findByText(/bob@/)

    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    await waitFor(() => {
      const calls = fetchMock.mock.calls.filter(([input]) => requestUrl(input as string | URL | Request).startsWith('/listUsers'))
      expect(calls.length).toBeGreaterThanOrEqual(2)
    })
    const lastListUsersCall = fetchMock.mock.calls.filter(([input]) => requestUrl(input as string | URL | Request).startsWith('/listUsers')).at(-1)
    expect(requestUrl(lastListUsersCall![0] as string | URL | Request)).not.toContain('q=')
  })
})

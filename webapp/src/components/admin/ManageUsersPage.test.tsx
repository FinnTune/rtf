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
})

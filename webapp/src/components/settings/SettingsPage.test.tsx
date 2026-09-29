import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { StatusBanner } from '../common/StatusBanner'
import { SettingsPage } from './SettingsPage'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function checkLoginResponse(email = 'actual@example.com') {
  return new Response(
    JSON.stringify({ loggedIn: true, id: 42, username: 'actual_user', email, joined: '2026-01-01', otp: 'x', role: 'user' }),
    { status: 200 },
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('SettingsPage', () => {
  it('loads and pre-fills the current profile', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse()
      if (url.startsWith('/profile')) {
        return new Response(JSON.stringify({ fname: 'Actual', lname: 'User', email: 'actual@example.com' }), { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <SettingsPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    expect(await screen.findByDisplayValue('Actual')).toBeInTheDocument()
    expect(screen.getByDisplayValue('User')).toBeInTheDocument()
    expect(screen.getByDisplayValue('actual@example.com')).toBeInTheDocument()
  })

  it('submits the updated profile and re-checks login so AuthContext picks up the new email', async () => {
    let checkLoginCalls = 0
    let updateProfileBody: unknown
    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) {
        checkLoginCalls += 1
        return checkLoginResponse(checkLoginCalls === 1 ? 'actual@example.com' : 'new@example.com')
      }
      if (url.startsWith('/profile')) {
        return new Response(JSON.stringify({ fname: 'Actual', lname: 'User', email: 'actual@example.com' }), { status: 200 })
      }
      if (url.startsWith('/updateProfile')) {
        updateProfileBody = JSON.parse(init!.body as string)
        return new Response('Profile updated', { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <StatusBanner />
          <SettingsPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    const emailInput = await screen.findByDisplayValue('actual@example.com')
    await userEvent.clear(emailInput)
    await userEvent.type(emailInput, 'new@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Save Profile' }))

    await waitFor(() => expect(screen.getByText('Profile updated.')).toBeInTheDocument())
    expect(updateProfileBody).toEqual({ fname: 'Actual', lname: 'User', email: 'new@example.com' })
    // Proves refresh() (re-running checkLogin) actually happened, not just
    // that the PATCH itself succeeded.
    expect(checkLoginCalls).toBe(2)
  })

  it('shows the server error when the new email is already in use', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse()
      if (url.startsWith('/profile')) {
        return new Response(JSON.stringify({ fname: 'Actual', lname: 'User', email: 'actual@example.com' }), { status: 200 })
      }
      if (url.startsWith('/updateProfile')) {
        return new Response('That email is already in use', { status: 409 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <StatusBanner />
          <SettingsPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    await screen.findByDisplayValue('actual@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Save Profile' }))

    expect(await screen.findByText('Err: That email is already in use')).toBeInTheDocument()
  })

  it('rejects submitting the password form when the new passwords do not match, without calling the server', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse()
      if (url.startsWith('/profile')) {
        return new Response(JSON.stringify({ fname: 'Actual', lname: 'User', email: 'actual@example.com' }), { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <StatusBanner />
          <SettingsPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    await screen.findByDisplayValue('actual@example.com')
    await userEvent.type(screen.getByLabelText('Current Password:'), 'oldpassword1')
    await userEvent.type(screen.getByLabelText('New Password:'), 'newpassword1')
    await userEvent.type(screen.getByLabelText('Confirm New Password:'), 'different1')
    await userEvent.click(screen.getByRole('button', { name: 'Change Password' }))

    expect(await screen.findByText('New passwords do not match')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([input]) => requestUrl(input as string | URL | Request).startsWith('/updatePassword'))).toBe(
      false,
    )
  })

  it('submits a matching password change and clears the form on success', async () => {
    let updatePasswordBody: unknown
    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse()
      if (url.startsWith('/profile')) {
        return new Response(JSON.stringify({ fname: 'Actual', lname: 'User', email: 'actual@example.com' }), { status: 200 })
      }
      if (url.startsWith('/updatePassword')) {
        updatePasswordBody = JSON.parse(init!.body as string)
        return new Response('Password updated', { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <StatusBanner />
          <SettingsPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    await screen.findByDisplayValue('actual@example.com')
    await userEvent.type(screen.getByLabelText('Current Password:'), 'oldpassword1')
    await userEvent.type(screen.getByLabelText('New Password:'), 'newpassword1')
    await userEvent.type(screen.getByLabelText('Confirm New Password:'), 'newpassword1')
    await userEvent.click(screen.getByRole('button', { name: 'Change Password' }))

    await waitFor(() => expect(screen.getByText('Password updated.')).toBeInTheDocument())
    expect(updatePasswordBody).toEqual({ current_password: 'oldpassword1', new_password: 'newpassword1' })
    expect(screen.getByLabelText('Current Password:')).toHaveValue('')
    expect(screen.getByLabelText('New Password:')).toHaveValue('')
    expect(screen.getByLabelText('Confirm New Password:')).toHaveValue('')
  })

  it('shows the server error when the current password is wrong, without clearing the form', async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) return checkLoginResponse()
      if (url.startsWith('/profile')) {
        return new Response(JSON.stringify({ fname: 'Actual', lname: 'User', email: 'actual@example.com' }), { status: 200 })
      }
      if (url.startsWith('/updatePassword')) {
        return new Response('Current password is incorrect', { status: 400 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <StatusBanner />
          <SettingsPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    await screen.findByDisplayValue('actual@example.com')
    await userEvent.type(screen.getByLabelText('Current Password:'), 'wrongpassword')
    await userEvent.type(screen.getByLabelText('New Password:'), 'newpassword1')
    await userEvent.type(screen.getByLabelText('Confirm New Password:'), 'newpassword1')
    await userEvent.click(screen.getByRole('button', { name: 'Change Password' }))

    expect(await screen.findByText('Err: Current password is incorrect')).toBeInTheDocument()
    expect(screen.getByLabelText('Current Password:')).toHaveValue('wrongpassword')
  })
})

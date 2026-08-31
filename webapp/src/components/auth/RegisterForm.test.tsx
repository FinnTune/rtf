import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { StatusBanner } from '../common/StatusBanner'
import { RegisterForm } from './RegisterForm'

function renderRegisterForm() {
  const onSwitchToLogin = vi.fn()
  const onRegistered = vi.fn()
  render(
    <StatusMessageProvider>
      <AuthProvider>
        <StatusBanner />
        <RegisterForm onSwitchToLogin={onSwitchToLogin} onRegistered={onRegistered} />
      </AuthProvider>
    </StatusMessageProvider>,
  )
  return { onSwitchToLogin, onRegistered }
}

async function fillRequiredFields(password: string, confirmPassword: string) {
  await userEvent.type(screen.getByLabelText('First Name:'), 'Ada')
  await userEvent.type(screen.getByLabelText('Last Name:'), 'Lovelace')
  await userEvent.type(screen.getByLabelText('Username:'), 'ada')
  await userEvent.type(screen.getByLabelText('Email:'), 'ada@example.com')
  await userEvent.type(screen.getByLabelText('Age:'), '30')
  await userEvent.type(screen.getByLabelText('Password:'), password)
  await userEvent.type(screen.getByLabelText('Confirm Password:'), confirmPassword)
}

function loggedOutCheckLoginResponse() {
  return new Response(JSON.stringify({ loggedIn: false }), { status: 200 })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('RegisterForm', () => {
  it('rejects a mismatched confirm-password without ever calling the server', async () => {
    const fetchMock = vi.fn().mockResolvedValue(loggedOutCheckLoginResponse())
    vi.stubGlobal('fetch', fetchMock)
    renderRegisterForm()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/checkLogin', expect.anything()))
    fetchMock.mockClear()

    await fillRequiredFields('password123', 'somethingElse')
    await userEvent.click(screen.getByRole('button', { name: 'Register' }))

    expect(await screen.findByText('Passwords do not match')).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('registers successfully and hands control back to the login view', async () => {
    const fetchMock = vi.fn().mockResolvedValue(loggedOutCheckLoginResponse())
    vi.stubGlobal('fetch', fetchMock)
    const { onRegistered } = renderRegisterForm()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/checkLogin', expect.anything()))

    fetchMock.mockResolvedValue(new Response('Registration successful.', { status: 200 }))
    await fillRequiredFields('password123', 'password123')
    await userEvent.click(screen.getByRole('button', { name: 'Register' }))

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/register',
        expect.objectContaining({
          body: JSON.stringify({
            fname: 'Ada',
            lname: 'Lovelace',
            uname: 'ada',
            email: 'ada@example.com',
            age: '30',
            gender: 'male',
            password: 'password123',
          }),
        }),
      ),
    )
    expect(await screen.findByText('You are now registered. Please login.')).toBeInTheDocument()
    expect(onRegistered).toHaveBeenCalledOnce()
  })

  it('shows the server error message when registration fails (e.g. duplicate username)', async () => {
    const fetchMock = vi.fn().mockResolvedValue(loggedOutCheckLoginResponse())
    vi.stubGlobal('fetch', fetchMock)
    renderRegisterForm()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/checkLogin', expect.anything()))

    fetchMock.mockResolvedValue(new Response('Username or email already exists', { status: 409 }))
    await fillRequiredFields('password123', 'password123')
    await userEvent.click(screen.getByRole('button', { name: 'Register' }))

    expect(await screen.findByText(/Username or email already exists/)).toBeInTheDocument()
  })
})

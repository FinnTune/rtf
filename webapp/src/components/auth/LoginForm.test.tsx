import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { StatusBanner } from '../common/StatusBanner'
import { LoginForm } from './LoginForm'

function renderLoginForm() {
  const onSwitchToRegister = vi.fn()
  render(
    <StatusMessageProvider>
      <AuthProvider>
        <StatusBanner />
        <LoginForm onSwitchToRegister={onSwitchToRegister} />
      </AuthProvider>
    </StatusMessageProvider>,
  )
  return { onSwitchToRegister }
}

function loggedOutCheckLoginResponse() {
  return new Response(JSON.stringify({ loggedIn: false }), { status: 200 })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('LoginForm', () => {
  it('submits the entered credentials to /login', async () => {
    const fetchMock = vi.fn().mockResolvedValue(loggedOutCheckLoginResponse())
    vi.stubGlobal('fetch', fetchMock)
    renderLoginForm()
    // Wait out AuthProvider's initial /checkLogin before it can be confused
    // with the /login call this test asserts on.
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/checkLogin', expect.anything()))

    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({ loggedIn: false, id: 1, username: 'alice', email: 'a@example.com', joined: '2026-01-01', otp: 'x' }),
        { status: 200 },
      ),
    )

    await userEvent.type(screen.getByLabelText('Login'), 'alice')
    await userEvent.type(screen.getByLabelText('Password'), 'password123')
    await userEvent.click(screen.getByRole('button', { name: 'Login' }))

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/login',
        expect.objectContaining({ body: JSON.stringify({ username: 'alice', password: 'password123' }) }),
      ),
    )
  })

  it('shows the server error message on a failed login', async () => {
    const fetchMock = vi.fn().mockResolvedValue(loggedOutCheckLoginResponse())
    vi.stubGlobal('fetch', fetchMock)
    renderLoginForm()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/checkLogin', expect.anything()))

    fetchMock.mockResolvedValue(new Response('', { status: 401 }))

    await userEvent.type(screen.getByLabelText('Login'), 'alice')
    await userEvent.type(screen.getByLabelText('Password'), 'wrongpassword')
    await userEvent.click(screen.getByRole('button', { name: 'Login' }))

    expect(await screen.findByText(/Request failed \(401\)/)).toBeInTheDocument()
    // The button must be re-enabled after a failed attempt, not stuck
    // showing "Logging in...".
    expect(screen.getByRole('button', { name: 'Login' })).toBeEnabled()
  })

  it('calls onSwitchToRegister when "Go to registration" is clicked', async () => {
    const fetchMock = vi.fn().mockResolvedValue(loggedOutCheckLoginResponse())
    vi.stubGlobal('fetch', fetchMock)
    const { onSwitchToRegister } = renderLoginForm()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/checkLogin', expect.anything()))

    await userEvent.click(screen.getByRole('button', { name: 'Go to registration' }))
    expect(onSwitchToRegister).toHaveBeenCalledOnce()
  })
})

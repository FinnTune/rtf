import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider, useAuth } from './AuthContext'

function mockFetchOnce(response: Response) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
}

const loggedOutResponse = () => new Response(JSON.stringify({ loggedIn: false }), { status: 200 })
const loggedInResponse = () =>
  new Response(
    JSON.stringify({ loggedIn: true, id: 1, username: 'alice', email: 'alice@example.com', joined: '2026-01-01', otp: 'otp-1' }),
    { status: 200 },
  )

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('AuthProvider', () => {
  it('starts loading, then resolves to no user when /checkLogin reports logged out', async () => {
    mockFetchOnce(loggedOutResponse())
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider })

    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.user).toBeNull()
  })

  it('resolves to the authenticated user when /checkLogin reports logged in', async () => {
    mockFetchOnce(loggedInResponse())
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider })

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.user).toEqual({
      id: 1,
      username: 'alice',
      email: 'alice@example.com',
      joined: '2026-01-01',
      otp: 'otp-1',
    })
  })

  it('treats a network error on the initial check as logged out, not a crash', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider })

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.user).toBeNull()
  })

  it('login() sets the user on success', async () => {
    mockFetchOnce(loggedOutResponse())
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider })
    await waitFor(() => expect(result.current.loading).toBe(false))

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(loggedInResponse()))
    await act(async () => {
      await result.current.login('alice', 'password123')
    })
    expect(result.current.user?.username).toBe('alice')
  })

  it('logout() clears the user', async () => {
    mockFetchOnce(loggedInResponse())
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider })
    await waitFor(() => expect(result.current.user).not.toBeNull())

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ loggedIn: false }), { status: 200 })))
    await act(async () => {
      await result.current.logout()
    })
    expect(result.current.user).toBeNull()
  })
})

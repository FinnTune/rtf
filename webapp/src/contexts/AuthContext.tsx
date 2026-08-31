import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import * as authApi from '../api/auth'
import type { AuthUser, RegisterPayload } from '../types'

interface AuthContextValue {
  user: AuthUser | null
  // True only until the initial /checkLogin call resolves.
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  register: (payload: RegisterPayload) => Promise<void>
  logout: () => Promise<void>
  // Re-verifies the session and mints a fresh OTP (OTPs are single-use and
  // short-lived — nothing should ever hold onto and reuse one). Updates
  // `user` from the result (e.g. to null if the session expired) and
  // returns it, so a caller like the WebSocket reconnect flow can branch on
  // it directly.
  refresh: () => Promise<AuthUser | null>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    authApi
      .checkLogin()
      .then((result) => {
        if (!cancelled) setUser(result)
      })
      .catch(() => {
        if (!cancelled) setUser(null)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const authUser = await authApi.login(username, password)
    setUser(authUser)
  }, [])

  const register = useCallback(async (payload: RegisterPayload) => {
    await authApi.register(payload)
  }, [])

  const logout = useCallback(async () => {
    await authApi.logout()
    setUser(null)
  }, [])

  const refresh = useCallback(async () => {
    const result = await authApi.checkLogin()
    setUser(result)
    return result
  }, [])

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout, refresh }}>{children}</AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return ctx
}

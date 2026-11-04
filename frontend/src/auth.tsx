import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, tokens, type User } from './api'

interface AuthState {
  user: User | null
  loading: boolean
  signIn: (email: string, password: string) => Promise<void>
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!tokens.access && !tokens.refresh) { setLoading(false); return }
    api.me()
      .then((r) => setUser(r.user))
      .catch(() => tokens.clear())
      .finally(() => setLoading(false))
  }, [])

  const signIn = useCallback(async (email: string, password: string) => {
    const res = await api.login(email, password)
    tokens.set(res.access_token, res.refresh_token)
    setUser(res.user)
  }, [])

  const signOut = useCallback(async () => {
    try { await api.logout() } catch { /* best effort */ }
    tokens.clear()
    setUser(null)
  }, [])

  const value = useMemo(() => ({ user, loading, signIn, signOut }), [user, loading, signIn, signOut])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth outside provider')
  return ctx
}

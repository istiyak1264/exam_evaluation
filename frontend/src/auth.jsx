import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { api, tokens } from './api'


const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!tokens.access && !tokens.refresh) { setLoading(false); return }
    api.me()
      .then((r) => setUser(r.user))
      .catch(() => tokens.clear())
      .finally(() => setLoading(false))
  }, [])

  const signIn = useCallback(async (email, password) => {
    const res = await api.login(email, password)
    tokens.set(res.access_token, res.refresh_token)
    setUser(res.user)
  }, [])

  const signUp = useCallback(async (payload) => {
    const res = await api.register(payload)
    tokens.set(res.access_token, res.refresh_token)
    setUser(res.user)
    return res.user
  }, [])

  const signOut = useCallback(async () => {
    try { await api.logout() } catch { /* best effort */ }
    tokens.clear()
    setUser(null)
  }, [])

  const value = useMemo(() => ({ user, loading, signIn, signUp, signOut }), [user, loading, signIn, signUp, signOut])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth outside provider')
  return ctx
}

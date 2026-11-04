import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { Note } from '../components/ui'

export default function Login() {
  const { signIn } = useAuth()
  const nav = useNavigate()
  const loc = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await signIn(email.trim(), password)
      nav(loc.state?.from ?? '/', { replace: true })
    } catch (err) {
      setError((err).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-wrap">
      <form className="login-card stack" onSubmit={submit}>
        <div className="login-brand">
          <div className="brand-mark">Script Evaluation System</div>
          <small>Anonymised double-blind marking</small>
        </div>
        {error && <Note kind="danger">{error}</Note>}
        <div className="field">
          <label htmlFor="email">Email</label>
          <input id="email" type="email" autoComplete="username" required value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="password">Password</label>
          <input id="password" type="password" autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />
        </div>
        <button className="btn primary" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
        <p className="hint">
          Students: use the account issued by the examination office. Examiners never see names on the scripts they mark.
        </p>
      </form>
    </div>
  )
}

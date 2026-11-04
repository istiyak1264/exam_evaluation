import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { ErrorSummary, SkipLink } from '../components/gov'

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
      <SkipLink />
      <div className="login-bar">
        Script Evaluation System
        <small>Examination office</small>
      </div>
      <div className="login-body">
      <form className="login-card" onSubmit={submit}>
        <div className="login-brand">
          <div className="brand-mark">Sign in</div>
          <small>Anonymised double-blind marking</small>
        </div>
        <ErrorSummary errors={error ? [error] : []} />
        <div className="field">
          <label htmlFor="email">Email</label>
          <input id="email" type="email" autoComplete="username" required value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="password">Password</label>
          <input id="password" type="password" autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />
        </div>
        <button className="btn primary" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
        <p className="hint" style={{ marginTop: '1.2rem', marginBottom: 0 }}>
          Students: use the account issued by the examination office. Examiners only see the scripts assigned to them, without names.
        </p>
      </form>
      </div>
    </div>
  )
}

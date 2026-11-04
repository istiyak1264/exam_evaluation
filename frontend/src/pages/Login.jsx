import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { AuthShell } from '../components/AuthShell'
import { ErrorSummary } from '../components/gov'
import PasswordField from '../components/PasswordField'

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
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthShell
      title="Sign in"
      intro="Use the account issued to you, or one you created. Examiners only ever see the cover-sheet code of the scripts assigned to them."
      footer={<>No account yet? <a href="/signup">Create one</a></>}
    >
      <form onSubmit={submit}>
        <ErrorSummary errors={error ? [error] : []} />
        <div className="field">
          <label htmlFor="email">Email</label>
          <input id="email" type="email" autoComplete="username" required value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <PasswordField id="password" label="Password" value={password} onChange={(e) => setPassword(e.target.value)} />
        <button className="btn primary" style={{ width: '100%' }} disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
      </form>
    </AuthShell>
  )
}

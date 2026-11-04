import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { AuthShell } from '../components/AuthShell'
import { ErrorSummary } from '../components/gov'
import PasswordField from '../components/PasswordField'

const ROLES = [
  { value: 'STUDENT', label: 'Student', hint: 'See your own scripts and results.' },
  { value: 'EVALUATOR', label: 'Evaluator', hint: 'Mark the scripts the board assigns to you.' },
]

export default function Signup() {
  const { signUp } = useAuth()
  const nav = useNavigate()
  const [form, setForm] = useState({ full_name: '', email: '', password: '', role: 'STUDENT', student_no: '' })
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  const set = (k) => (e) => setForm({ ...form, [k]: e.target.value })

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const user = await signUp({ ...form, role: form.role })
      nav(user.role === 'EVALUATOR' ? '/queue' : '/me', { replace: true })
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthShell
      title="Create an account"
      intro="Students and external examiners may register here. Administrator accounts are created by the examination board."
      footer={<>Already registered? <a href="/login">Sign in</a></>}
    >
      <form onSubmit={submit}>
        <ErrorSummary errors={error ? [error] : []} />

        <fieldset className="field" style={{ border: 0, padding: 0, maxWidth: 'none' }}>
          <legend className="field-legend">Register as</legend>
          <div className="choice-row">
            {ROLES.map((r) => (
              <label key={r.value} className={`choice${form.role === r.value ? ' selected' : ''}`}>
                <input type="radio" name="role" value={r.value} checked={form.role === r.value} onChange={set('role')} />
                <span>
                  <strong>{r.label}</strong>
                  <span className="hint">{r.hint}</span>
                </span>
              </label>
            ))}
          </div>
        </fieldset>

        <div className="field">
          <label htmlFor="full_name">Full name</label>
          <input id="full_name" required minLength={3} autoComplete="name" value={form.full_name} onChange={set('full_name')} />
        </div>
        <div className="field">
          <label htmlFor="email">Email</label>
          <input id="email" type="email" required autoComplete="username" value={form.email} onChange={set('email')} />
        </div>
        <PasswordField
          id="password"
          label="Password"
          hint="At least 10 characters."
          hintId="pw-hint"
          autoComplete="new-password"
          minLength={10}
          value={form.password}
          onChange={set('password')}
        />

        {form.role === 'STUDENT' && (
          <div className="field">
            <label htmlFor="student_no">Matriculation number <span className="hint">(optional)</span></label>
            <span className="hint" id="no-hint">If the board already holds your record, this links it so your scripts appear straight away.</span>
            <input id="student_no" aria-describedby="no-hint" value={form.student_no} onChange={set('student_no')} placeholder="e.g. MAT/2026/0101" />
          </div>
        )}

        <button className="btn primary" style={{ width: '100%' }} disabled={busy}>{busy ? 'Creating account…' : 'Create account'}</button>
      </form>
    </AuthShell>
  )
}

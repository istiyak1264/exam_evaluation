import { useState } from 'react'
import { api } from '../api'
import { useAuth } from '../auth'
import { Note } from '../components/ui'

export default function Account() {
  const { user } = useAuth()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [msg, setMsg] = useState<string | null>(null)
  const [err, setErr] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr(null)
    setMsg(null)
    try {
      await api.changePassword(current, next)
      setMsg('Password changed.')
      setCurrent('')
      setNext('')
    } catch (e2) {
      setErr((e2 as Error).message)
    }
  }

  return (
    <div className="stack" style={{ maxWidth: 480 }}>
      <div className="page-head"><div><h1>Account</h1></div></div>
      <div className="card stack">
        <div><span className="hint">Name</span><div>{user?.full_name}</div></div>
        <div><span className="hint">Email</span><div className="mono">{user?.email}</div></div>
        <div><span className="hint">Role</span><div>{user?.role.replace('_', ' ').toLowerCase()}</div></div>
      </div>
      {msg && <Note kind="info">{msg}</Note>}
      {err && <Note kind="danger">{err}</Note>}
      <form className="card stack" onSubmit={submit}>
        <h2>Change password</h2>
        <div className="field"><label>Current password</label><input type="password" required value={current} onChange={(e) => setCurrent(e.target.value)} /></div>
        <div className="field"><label>New password (min 10 chars)</label><input type="password" minLength={10} required value={next} onChange={(e) => setNext(e.target.value)} /></div>
        <button className="btn primary">Update password</button>
      </form>
    </div>
  )
}

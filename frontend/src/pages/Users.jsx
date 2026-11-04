import { useState } from 'react'
import { api, when } from '../api'
import { Empty, Loading, Note, useFetch } from '../components/ui'

const ROLES = ['ADMIN', 'HALL_MANAGER', 'EVALUATOR', 'STUDENT']

export default function Users() {
  const [role, setRole] = useState('')
  const { data, error, loading, reload } = useFetch(() => api.users(role), [role])
  const [form, setForm] = useState({ email: '', full_name: '', role: 'EVALUATOR', password: '' })
  const [msg, setMsg] = useState(null)

  async function create(e) {
    e.preventDefault()
    setMsg(null)
    try {
      await api.createUser(form)
      setMsg(`${form.full_name} created.`)
      setForm({ email: '', full_name: '', role: 'EVALUATOR', password: '' })
      reload()
    } catch (err) {
      setMsg((err).message)
    }
  }

  async function toggle(id, active) {
    try { await api.patchUser(id, { is_active: active }); reload() } catch (e) { setMsg((e).message) }
  }

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>People</h1>
          <p>Accounts for the examination office, examiners and students. Deactivating keeps history intact.</p>
        </div>
        <select value={role} onChange={(e) => setRole(e.target.value)} style={{ padding: '.5rem', borderRadius: 8, border: '1px solid var(--line)' }}>
          <option value="">All roles</option>
          {ROLES.map((r) => <option key={r} value={r}>{r.replace('_', ' ').toLowerCase()}</option>)}
        </select>
      </div>

      {msg && <Note kind="info">{msg}</Note>}
      {error && <Note kind="danger">{error}</Note>}

      <form className="card grid" onSubmit={create}>
        <div className="field"><label>Full name</label><input required value={form.full_name} onChange={(e) => setForm({ ...form, full_name: e.target.value })} /></div>
        <div className="field"><label>Email</label><input type="email" required value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></div>
        <div className="field">
          <label>Role</label>
          <select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value})}>
            {ROLES.map((r) => <option key={r} value={r}>{r.replace('_', ' ').toLowerCase()}</option>)}
          </select>
        </div>
        <div className="field"><label>Temporary password</label><input type="text" minLength={10} required value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} /></div>
        <div className="field" style={{ justifyContent: 'end' }}><button className="btn primary">Create account</button></div>
      </form>

      {loading && <Loading />}
      {data && data.users.length === 0 && <Empty>No accounts.</Empty>}
      {data && data.users.length > 0 && (
        <div className="card">
          <table>
            <thead><tr><th>Name</th><th>Email</th><th>Role</th><th>Status</th><th>Created</th><th /></tr></thead>
            <tbody>
              {data.users.map((u) => (
                <tr key={u.id}>
                  <td>{u.full_name}</td>
                  <td className="mono">{u.email}</td>
                  <td><span className="badge info">{u.role.replace('_', ' ').toLowerCase()}</span></td>
                  <td><span className={`badge ${u.is_active ? 'ok' : 'danger'}`}>{u.is_active ? 'active' : 'disabled'}</span></td>
                  <td className="hint">{when(u.created_at)}</td>
                  <td><button className="btn sm" onClick={() => void toggle(u.id, !u.is_active)}>{u.is_active ? 'Disable' : 'Enable'}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

import { useAuth } from '../auth'

export default function Account() {
  const { user } = useAuth()

  return (
    <div className="stack" style={{ maxWidth: 480 }}>
      <div className="page-head">
        <div>
          <h1>Account</h1>
          <p>The password was set by the examination office when the account was created.</p>
        </div>
      </div>
      <div className="card stack">
        <div><span className="hint">Name</span><div>{user?.full_name}</div></div>
        <div><span className="hint">Email</span><div className="mono">{user?.email}</div></div>
        <div><span className="hint">Role</span><div>{user?.role.replace('_', ' ').toLowerCase()}</div></div>
      </div>
    </div>
  )
}

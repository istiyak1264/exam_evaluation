import { api, when } from '../api'
import { Empty, Loading, Note, useFetch } from '../components/ui'

export default function Audit() {
  const { data, error, loading } = useFetch(() => api.audit(), [])

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Audit trail</h1>
          <p>Append-only record of registrations, appointments, marks and overrides. Newest first.</p>
        </div>
      </div>

      {error && <Note kind="danger">{error}</Note>}
      {loading && <Loading />}
      {data && data.entries.length === 0 && <Empty>Nothing recorded yet.</Empty>}

      {data && data.entries.length > 0 && (
        <div className="card">
          <table>
            <thead><tr><th>When</th><th>Actor</th><th>Action</th><th>Entity</th><th>Detail</th></tr></thead>
            <tbody>
              {data.entries.map((e) => (
                <tr key={e.id}>
                  <td className="hint">{when(e.created_at)}</td>
                  <td>{e.actor_name ?? 'system'}<div className="hint">{e.actor_role ?? ''}</div></td>
                  <td className="mono">{e.action}</td>
                  <td className="hint">{e.entity}</td>
                  <td className="mono" style={{ fontSize: '.75rem', maxWidth: 320, wordBreak: 'break-all' }}>{JSON.stringify(e.meta)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

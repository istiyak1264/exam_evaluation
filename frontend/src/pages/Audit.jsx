import { useState } from 'react'
import { api, whenBST } from '../api'
import { Empty, Loading, Note, useFetch } from '../components/ui'

export default function Audit() {
  const { data, error, loading, reload } = useFetch(() => api.audit(), [])
  const [msg, setMsg] = useState(null)
  const [busy, setBusy] = useState(false)

  async function clearLogs() {
    if (!window.confirm('Clear the whole audit trail? This cannot be undone.')) return
    setBusy(true)
    setMsg(null)
    try {
      const res = await api.clearAudit()
      setMsg(`Audit trail cleared — ${res.cleared} entr${res.cleared === 1 ? 'y' : 'ies'} removed.`)
      reload()
    } catch (e) {
      setMsg(e.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Audit trail</h1>
          <p>Append-only record of registrations, appointments, marks and overrides. Newest first. Times are Bangladesh Standard Time (UTC+6).</p>
        </div>
        <button className="btn danger" onClick={() => void clearLogs()} disabled={busy || !data?.entries.length}>
          {busy ? 'Clearing…' : 'Clear all logs'}
        </button>
      </div>

      {msg && <Note kind="info">{msg}</Note>}
      {error && <Note kind="danger">{error}</Note>}
      {loading && <Loading />}
      {data && data.entries.length === 0 && <Empty>Nothing recorded yet.</Empty>}

      {data && data.entries.length > 0 && (
        <div className="card">
          <table>
            <thead><tr><th>When (BST)</th><th>Actor</th><th>Action</th><th>Entity</th><th>Detail</th></tr></thead>
            <tbody>
              {data.entries.map((e) => (
                <tr key={e.id}>
                  <td className="hint">{whenBST(e.created_at)}</td>
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

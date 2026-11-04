import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, money, when } from '../api'
import { Badge, Empty, Grade, Loading, Note } from '../components/ui'

// Search retrieves marks by any of: name, matriculation number, session, script code.
export default function Search() {
  const [q, setQ] = useState('')
  const [sessionId, setSessionId] = useState('')
  const [status, setStatus] = useState('')
  const [sessions, setSessions] = useState([])
  const [rows, setRows] = useState(null)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => { api.sessions().then((r) => setSessions(r.sessions)).catch(() => undefined) }, [])

  async function run(e) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const res = await api.results({ q, session_id: sessionId, status })
      setRows(res.results)
    } catch (err) {
      setError((err).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Find results</h1>
          <p>Search by student name, matriculation number, academic session or the script's cover-sheet code.</p>
        </div>
        <a className="btn" href={`/api/export/results.csv?q=${encodeURIComponent(q)}`}>Export CSV</a>
      </div>

      {error && <Note kind="danger">{error}</Note>}

      <form className="card row" onSubmit={run}>
        <input
          value={q} onChange={(e) => setQ(e.target.value)} placeholder="Name, matriculation number, session or code"
          style={{ flex: 2, minWidth: 240, padding: '.55rem .7rem', border: '1px solid var(--line)', borderRadius: 8 }}
        />
        <select value={sessionId} onChange={(e) => setSessionId(e.target.value)} style={{ padding: '.55rem', borderRadius: 8, border: '1px solid var(--line)' }}>
          <option value="">Any session</option>
          {sessions.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
        </select>
        <select value={status} onChange={(e) => setStatus(e.target.value)} style={{ padding: '.55rem', borderRadius: 8, border: '1px solid var(--line)' }}>
          <option value="">Any stage</option>
          {['EVAL_PENDING', 'THIRD_REVIEW', 'RESOLVED'].map((s) => <option key={s} value={s}>{s.replace(/_/g, ' ').toLowerCase()}</option>)}
        </select>
        <button className="btn primary" disabled={busy}>{busy ? 'Searching…' : 'Search'}</button>
      </form>

      {busy && <Loading />}
      {rows && rows.length === 0 && <Empty>No matching scripts.</Empty>}

      {rows && rows.length > 0 && (
        <div className="card">
          <table>
            <thead><tr><th>Code</th><th>Student</th><th>Session</th><th>Course</th><th>Stage</th><th>Final</th><th>Grade</th><th>Challenge</th><th /></tr></thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.script_id}>
                  <td className="mono">{r.script_code}</td>
                  <td>{r.student_name}<div className="hint mono">{r.student_no}</div></td>
                  <td className="hint">{r.session}</td>
                  <td>{r.course_code}</td>
                  <td><Badge status={r.status} /></td>
                  <td className="mono">{money(r.final_marks)}{r.passed === false && <div className="badge danger">fail</div>}</td>
                  <td><Grade grade={r.grade} /></td>
                  <td>{r.challenge_status ? <Badge status={r.challenge_status} /> : <span className="hint">—</span>}</td>
                  <td><Link className="btn sm" to={`/scripts/${r.script_id}`}>Open</Link></td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="hint">{rows.length} result(s) · last resolved {when(rows[0]?.resolved_at)}</p>
        </div>
      )}
    </div>
  )
}

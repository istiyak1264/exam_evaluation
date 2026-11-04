import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api, money, when } from '../api'
import { Badge, Empty, Loading, Note, useFetch } from '../components/ui'

export default function Scripts() {
  const [status, setStatus] = useState('')
  const [q, setQ] = useState('')
  const [term, setTerm] = useState('')
  const { data, error, loading } = useFetch(() => api.scripts({ status, q: term }), [status, term])

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Scripts</h1>
          <p>Every registered script, its cover-sheet code and where it sits in the workflow.</p>
        </div>
        <Link className="btn primary" to="/scripts/new">Register script</Link>
      </div>

      <form className="card row" onSubmit={(e) => { e.preventDefault(); setTerm(q) }}>
        <input
          placeholder="Code, student name or matriculation number"
          value={q} onChange={(e) => setQ(e.target.value)} style={{ flex: 1, minWidth: 220, padding: '.5rem .7rem', border: '1px solid var(--line)', borderRadius: 8 }}
        />
        <select value={status} onChange={(e) => setStatus(e.target.value)} style={{ padding: '.5rem', borderRadius: 8, border: '1px solid var(--line)' }}>
          <option value="">All stages</option>
          {['REGISTERED', 'ASSIGNED', 'EVAL_PENDING', 'THIRD_REVIEW', 'RESOLVED'].map((s) => (
            <option key={s} value={s}>{s.replace(/_/g, ' ').toLowerCase()}</option>
          ))}
        </select>
        <button className="btn">Filter</button>
      </form>

      {error && <Note kind="danger">{error}</Note>}
      {loading && <Loading />}
      {data && data.scripts.length === 0 && <Empty>No scripts match.</Empty>}

      {data && data.scripts.length > 0 && (
        <div className="card">
          <table>
            <thead>
              <tr><th>Code</th><th>Course</th><th>Student</th><th>Stage</th><th>Marks</th><th>Registered</th><th /></tr>
            </thead>
            <tbody>
              {data.scripts.map((s) => (
                <tr key={s.id}>
                  <td className="mono">{s.script_code}</td>
                  <td>{s.course_code}<div className="hint">{s.course_title}</div></td>
                  <td>{s.student_name}<div className="hint mono">{s.student_no}</div></td>
                  <td><Badge status={s.status} /></td>
                  <td className="mono">{s.marks.map((m) => m.score).join(' / ') || '—'}</td>
                  <td className="hint">{when(s.registered_at)}</td>
                  <td><Link className="btn sm" to={`/scripts/${s.id}`}>Open</Link></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {data && <p className="hint">{data.scripts.length} shown · final marks are averaged below a 13-mark gap and mediated by a third examiner at or above it.</p>}
      {false && money(0)}
    </div>
  )
}

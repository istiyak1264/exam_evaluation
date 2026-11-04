import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api, when } from '../api'
import { Badge, Empty, Loading, Note, useFetch } from '../components/ui'

export default function Queue() {
  const [view, setView] = useState('PENDING')
  const { data, error, loading } = useFetch(() => api.queue(view), [view])

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>My marking</h1>
          <p>Scripts appointed to you. Identity is withheld — mark against the cover-sheet code only.</p>
        </div>
        <div className="row">
          <button className={`btn sm ${view === 'PENDING' ? 'primary' : ''}`} onClick={() => setView('PENDING')}>To mark</button>
          <button className={`btn sm ${view === 'MARKED' ? 'primary' : ''}`} onClick={() => setView('MARKED')}>Submitted</button>
          <button className={`btn sm ${view === '' ? 'primary' : ''}`} onClick={() => setView('')}>All</button>
        </div>
      </div>

      {error && <Note kind="danger">{error}</Note>}
      {loading && <Loading />}
      {data && data.queue.length === 0 && <Empty>Nothing here.</Empty>}

      {data && data.queue.length > 0 && (
        <div className="card">
          <table>
            <thead><tr><th>Code</th><th>Course</th><th>Slot</th><th>Stage</th><th>Assigned</th><th /></tr></thead>
            <tbody>
              {data.queue.map((s) => (
                <tr key={s.id}>
                  <td className="mono">{s.script_code}</td>
                  <td>{s.course_code}<div className="hint">{s.course_title}</div></td>
                  <td><span className="badge info">{s.my_slot.toLowerCase()}</span></td>
                  <td><Badge status={s.status} /></td>
                  <td className="hint">{when(s.assigned_at)}</td>
                  <td>
                    <Link className="btn sm" to={`/scripts/${s.id}`}>
                      {s.marks.length ? 'Review' : 'Mark'}
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

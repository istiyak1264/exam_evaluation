import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api, when } from '../api'
import { Badge, Empty, Loading, Note, useFetch } from '../components/ui'

export default function Challenges() {
  const [status, setStatus] = useState('OPEN')
  const { data, error, loading, reload } = useFetch(() => api.challenges(status), [status])
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [msg, setMsg] = useState<string | null>(null)

  async function decide(id: string, decision: 'UPHELD' | 'DISMISSED') {
    setMsg(null)
    try {
      await api.decideChallenge(id, decision, notes[id] ?? '')
      setMsg(`Challenge ${decision.toLowerCase()}.`)
      reload()
    } catch (e) {
      setMsg((e as Error).message)
    }
  }

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Grade challenges</h1>
          <p>Students can contest a published mark once. Upheld challenges go to the board for remarking.</p>
        </div>
        <div className="row">
          {['OPEN', 'UPHELD', 'DISMISSED', ''].map((s) => (
            <button key={s || 'all'} className={`btn sm ${status === s ? 'primary' : ''}`} onClick={() => setStatus(s)}>
              {s ? s.toLowerCase() : 'all'}
            </button>
          ))}
        </div>
      </div>

      {msg && <Note kind="info">{msg}</Note>}
      {error && <Note kind="danger">{error}</Note>}
      {loading && <Loading />}
      {data && data.challenges.length === 0 && <Empty>No challenges in this bucket.</Empty>}

      {data && data.challenges.map((c) => (
        <div className="card stack" key={c.id}>
          <div className="spread">
            <div>
              <strong>{c.student_name} <span className="mono hint">{c.student_no}</span></strong>
              <div className="hint">{c.course_code} · script <span className="mono">{c.script_code}</span> · {when(c.created_at)}</div>
            </div>
            <div className="row">
              <Badge status={c.status} />
              <Link className="btn sm" to={`/scripts/${c.script_id}`}>Open script</Link>
            </div>
          </div>
          <p style={{ margin: 0 }}>{c.reason}</p>
          {c.decision_note && <div className="hint">Decision note: {c.decision_note}</div>}
          {c.status === 'OPEN' && (
            <div className="row">
              <input
                placeholder="Decision note"
                value={notes[c.id] ?? ''}
                onChange={(e) => setNotes({ ...notes, [c.id]: e.target.value })}
                style={{ flex: 1, padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }}
              />
              <button className="btn accent sm" onClick={() => void decide(c.id, 'UPHELD')}>Uphold</button>
              <button className="btn sm" onClick={() => void decide(c.id, 'DISMISSED')}>Dismiss</button>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

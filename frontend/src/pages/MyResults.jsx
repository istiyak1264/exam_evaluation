import { useState } from 'react'
import { api, money, when } from '../api'
import { PdfPanel } from '../components/PdfPanel'
import { Badge, Empty, Grade, Loading, Note, useFetch } from '../components/ui'

export default function MyResults() {
  const { data, error, loading, reload } = useFetch(() => api.myResults(), [])
  const [open, setOpen] = useState(null)
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState(null)

  async function challenge(scriptId) {
    setBusy(true)
    setMsg(null)
    try {
      await api.createChallenge(scriptId, reason)
      setReason('')
      setMsg('Challenge submitted. The examination board will review your script.')
      reload()
    } catch (e) {
      setMsg((e).message)
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <Loading />
  if (error) return <Note kind="danger">{error}</Note>

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>My results</h1>
          {data && <p>{data.student.full_name} · <span className="mono">{data.student.student_no}</span> · {data.student.session}</p>}
        </div>
      </div>

      {msg && <Note kind="info">{msg}</Note>}
      <Note kind="info">
        Each script is marked independently by two examiners. Where their marks differ by 13 or more, a third examiner
        decides. You can open the marked script and raise a challenge on any published result.
      </Note>

      {data && data.results.length === 0 && <Empty>No scripts on record yet.</Empty>}

      {data && data.results.map((r) => (
        <div className="card stack" key={r.script_id}>
          <div className="spread">
            <div>
              <h2>{r.course_code} — {r.course_title}</h2>
              <div className="hint mono">{r.script_code} · registered {when(r.registered_at)}</div>
            </div>
            <div className="row">
              <Badge status={r.status} />
              {r.final_marks !== null && r.final_marks !== undefined
                ? <strong className="mono" style={{ fontSize: '1.3rem' }}>{money(r.final_marks)}</strong>
                : <span className="hint">pending</span>}
              <Grade grade={r.grade} />
            </div>
          </div>

          <div className="row">
            <button className="btn sm" onClick={() => setOpen(open === r.script_id ? null : r.script_id)}>
              {open === r.script_id ? 'Hide script' : 'View marked script'}
            </button>
            {r.challenge_status
              ? <span className="badge warn">challenge {r.challenge_status.toLowerCase()}</span>
              : r.final_marks !== null && r.final_marks !== undefined && (
                <span className="hint">Open the script, then raise a challenge below.</span>
              )}
          </div>

          {open === r.script_id && (
            <div className="stack">
              <PdfPanel scriptId={r.script_id} />
              {!r.challenge_status && r.final_marks !== null && (
                <form
                  className="stack"
                  onSubmit={(e) => { e.preventDefault(); void challenge(r.script_id) }}
                >
                  <div className="field">
                    <label>Why are you challenging this mark?</label>
                    <textarea required minLength={10} rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
                  </div>
                  <button className="btn primary" disabled={busy || reason.trim().length < 10}>Submit challenge</button>
                </form>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

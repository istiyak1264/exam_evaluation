import { useState } from 'react'
import { api, money, when } from '../api'
import { PdfPanel } from '../components/PdfPanel'
import { Breadcrumbs, SummaryList } from '../components/gov'
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
      setMsg('Challenge submitted. The board will review the script and, if upheld, your marks are released here.')
      reload()
    } catch (e) {
      setMsg(e.message)
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <Loading />
  if (error) return <Note kind="danger">{error}</Note>

  return (
    <div className="stack">
      <Breadcrumbs items={[{ label: 'My results' }]} />
      <div className="page-head">
        <div>
          <span className="caption">Candidate record</span>
          <h1>My results</h1>
          {data && <p>{data.student.full_name} · <span className="mono">{data.student.student_no}</span> · {data.student.session}</p>}
        </div>
      </div>

      {msg && <Note kind="info">{msg}</Note>}

      <Note kind="info">
        Scripts are marked twice, independently, and a third examiner is appointed where the marks differ by 13 or more.
        Marks are withheld until you challenge the evaluation and the board upholds it — then the marks and the total appear below.
      </Note>

      {data && data.results.length === 0 && <Empty>No scripts on record yet.</Empty>}

      {data && data.results.map((r) => (
        <div className="card stack" key={r.script_id}>
          <div className="spread">
            <div>
              <span className="caption">{r.script_code}</span>
              <h2>{r.course_code} — {r.course_title}</h2>
              <div className="hint">registered {when(r.registered_at)}</div>
            </div>
            <div className="row">
              <Badge status={r.status} />
              {r.revealed
                ? <><strong className="mono" style={{ fontSize: '1.3rem' }}>{money(r.final_marks)}</strong><Grade grade={r.grade} /></>
                : <span className="badge warn">marks withheld</span>}
            </div>
          </div>

          {r.revealed ? (
            <SummaryList
              rows={[
                ...(r.marks ?? []).map((m) => ({
                  key: `${m.slot.toLowerCase()} examiner`,
                  value: <><strong className="mono">{money(m.score)}</strong>{m.comment ? <div className="hint">{m.comment}</div> : null}</>,
                })),
                { key: 'Final marks', value: <><strong className="mono">{money(r.final_marks)}</strong><div className="hint">{r.passed ? 'Passed' : 'Below the pass mark'}</div></> },
              ]}
            />
          ) : (
            <p className="hint" style={{ margin: 0 }}>
              {r.challenge_status === 'OPEN'
                ? 'Your challenge is with the board. Marks appear here once it is upheld.'
                : r.challenge_status === 'DISMISSED'
                  ? 'Your challenge was dismissed. Marks stay withheld.'
                  : r.status === 'RESOLVED'
                    ? 'Review the script and raise a challenge to have the marks released.'
                    : 'Marking is still in progress.'}
            </p>
          )}

          <div className="row">
            <button className="btn sm secondary" onClick={() => setOpen(open === r.script_id ? null : r.script_id)}>
              {open === r.script_id ? 'Hide script' : 'View script'}
            </button>
            {r.challenge_status && <Badge status={r.challenge_status} />}
          </div>

          {open === r.script_id && (
            <div className="stack">
              <PdfPanel scriptId={r.script_id} />
              {!r.challenge_status && r.status === 'RESOLVED' && (
                <form className="stack" onSubmit={(e) => { e.preventDefault(); void challenge(r.script_id) }}>
                  <div className="field" style={{ maxWidth: 'none' }}>
                    <label htmlFor={`why-${r.script_id}`}>Why are you challenging this evaluation?</label>
                    <span className="hint">At least 10 characters. The board reads this before deciding.</span>
                    <textarea id={`why-${r.script_id}`} required minLength={10} rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
                  </div>
                  <button className="btn primary" disabled={busy || reason.trim().length < 10}>Submit challenge</button>
                </form>
              )}
              {r.challenge_status && <p className="hint" style={{ margin: 0 }}>One challenge per script.</p>}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

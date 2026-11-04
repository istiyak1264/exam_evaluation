import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, money, when } from '../api'
import { useAuth } from '../auth'
import { PdfPanel } from '../components/PdfPanel'
import { Badge, Grade, Loading, Note } from '../components/ui'

export default function ScriptDetail() {
  const { id = '' } = useParams()
  const { user } = useAuth()
  const isEvaluator = user?.role === 'EVALUATOR'
  const [script, setScript] = useState(null)
  const [blind, setBlind] = useState(null)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(true)
  const [evaluators, setEvaluators] = useState([])
  const [first, setFirst] = useState('')
  const [second, setSecond] = useState('')
  const [third, setThird] = useState('')
  const [score, setScore] = useState('')
  const [comment, setComment] = useState('')
  const [overrideScore, setOverrideScore] = useState('')
  const [overrideNote, setOverrideNote] = useState('')
  const [msg, setMsg] = useState(null)

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const res = await api.script(id)
      if (isEvaluator) setBlind(res)
      else setScript(res)
    } catch (e) {
      setError((e).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [id])

  useEffect(() => {
    if (isEvaluator) return
    api.users('EVALUATOR').then((r) => setEvaluators(r.users)).catch(() => setEvaluators([]))
  }, [isEvaluator])

  async function act(fn, note) {
    setError(null)
    setMsg(null)
    try {
      await fn()
      setMsg(note)
      await load()
    } catch (e) {
      setError((e).message)
    }
  }

  if (loading) return <Loading />
  if (error) return <Note kind="danger">{error}</Note>

  const marks = blind?.marks ?? script?.marks ?? []
  const status = blind?.status ?? script?.status ?? ''
  const maxMarks = blind?.max_marks ?? script?.max_marks ?? 100
  const final = script?.final_marks ?? blind?.final_marks ?? null
  const hasMyMark = isEvaluator && blind?.marks.some((m) => m.slot === blind.my_slot)

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1 className="mono">{blind?.script_code ?? script?.script_code}</h1>
          <p>
            {blind?.course_code ?? script?.course_code} · {blind?.course_title ?? script?.course_title} · paper out of {maxMarks}
            {!isEvaluator && script && <> · {script.page_count ?? '?'} page(s)</>}
          </p>
        </div>
        <div className="row">
          <Badge status={status} />
          <Link className="btn sm" to={isEvaluator ? '/queue' : '/scripts'}>Back</Link>
        </div>
      </div>

      {msg && <Note kind="info">{msg}</Note>}

      {isEvaluator && (
        <Note kind="info">
          Blinded view — no student name or matriculation number is shown. Your mark locks once submitted.
        </Note>
      )}

      <div className="detail-grid">
        <div className="card">
          <h2>Script pages</h2>
          <PdfPanel scriptId={id} />
        </div>

        <div className="stack">
          <div className="card stack">
            <h2>Marks</h2>
            {marks.length === 0 && <p className="hint">Nothing submitted yet.</p>}
            {marks.map((m) => (
              <div key={m.slot} className="row spread">
                <div>
                  <div className="hint">{m.slot.toLowerCase()} examiner{!isEvaluator && m.evaluator_name ? ` · ${m.evaluator_name}` : ''}</div>
                  {m.comment && <div style={{ fontSize: '.85rem' }}>{m.comment}</div>}
                  <div className="hint">{when(m.submitted_at)}</div>
                </div>
                <strong className="mono">{money(m.score)}</strong>
              </div>
            ))}
            {status === 'THIRD_REVIEW' && (
              <Note>A gap of 13 or more was recorded. A third external examiner must mark this script.</Note>
            )}
            {final !== null && (
              <div className="row spread" style={{ borderTop: '1px solid var(--line)', paddingTop: '.6rem' }}>
                <div>
                  <strong>Final marks</strong>
                  <div className="hint">{(script?.method ?? blind?.method)?.replace(/_/g, ' ').toLowerCase() ?? 'board'}</div>
                </div>
                <strong className="mono" style={{ fontSize: '1.2rem' }}>{money(final)}</strong>
              </div>
            )}
            {final !== null && script && <Grade grade={final >= 70 ? 'A' : final >= 60 ? 'B' : final >= 50 ? 'C' : final >= 45 ? 'D' : final >= 40 ? 'E' : 'F'} />}
          </div>

          {isEvaluator && !hasMyMark && (
            <form
              className="card stack"
              onSubmit={(e) => {
                e.preventDefault()
                void act(async () => {
                  await api.mark(id, Number(score), comment)
                  setScore('')
                  setComment('')
                }, 'Mark recorded.')
              }}
            >
              <h2>Submit your mark</h2>
              <div className="field">
                <label>Score out of {maxMarks}</label>
                <input type="number" step="0.01" min="0" max={maxMarks} required value={score} onChange={(e) => setScore(e.target.value)} />
              </div>
              <div className="field">
                <label>Comment (optional)</label>
                <textarea rows={3} value={comment} onChange={(e) => setComment(e.target.value)} />
              </div>
              <button className="btn accent">Submit mark</button>
            </form>
          )}

          {!isEvaluator && script && (
            <div className="card stack">
              <h2>Appointments</h2>
              {(status === 'REGISTERED' || status === 'ASSIGNED') && (
                <form
                  className="stack"
                  onSubmit={(e) => {
                    e.preventDefault()
                    void act(() => api.assign(id, first, second), 'Examiners appointed.')
                  }}
                >
                  <div className="field">
                    <label>First examiner (internal)</label>
                    <select value={first} onChange={(e) => setFirst(e.target.value)} required>
                      <option value="">Select…</option>
                      {evaluators.filter((e) => e.is_active).map((e) => <option key={e.id} value={e.id}>{e.full_name}</option>)}
                    </select>
                  </div>
                  <div className="field">
                    <label>Second examiner (external)</label>
                    <select value={second} onChange={(e) => setSecond(e.target.value)} required>
                      <option value="">Select…</option>
                      {evaluators.filter((e) => e.is_active && e.id !== first).map((e) => <option key={e.id} value={e.id}>{e.full_name}</option>)}
                    </select>
                  </div>
                  <button className="btn primary">Appoint both examiners</button>
                </form>
              )}
              {status === 'THIRD_REVIEW' && (
                <form
                  className="stack"
                  onSubmit={(e) => {
                    e.preventDefault()
                    void act(() => api.assignThird(id, third), 'Third examiner appointed.')
                  }}
                >
                  <div className="field">
                    <label>Third examiner (must be new)</label>
                    <select value={third} onChange={(e) => setThird(e.target.value)} required>
                      <option value="">Select…</option>
                      {evaluators.filter((e) => e.is_active && !script.marks.some((m) => m.evaluator_name === e.full_name)).map((e) => (
                        <option key={e.id} value={e.id}>{e.full_name}</option>
                      ))}
                    </select>
                  </div>
                  <button className="btn primary">Appoint third examiner</button>
                </form>
              )}
            </div>
          )}

          {!isEvaluator && script && (
            <div className="card stack">
              <h2>Board override</h2>
              <form
                className="stack"
                onSubmit={(e) => {
                  e.preventDefault()
                  void act(() => api.override(id, Number(overrideScore), overrideNote), 'Result overridden.')
                }}
              >
                <div className="field">
                  <label>Final marks</label>
                  <input type="number" step="0.01" min="0" max={maxMarks} required value={overrideScore} onChange={(e) => setOverrideScore(e.target.value)} />
                </div>
                <div className="field">
                  <label>Reason</label>
                  <input required value={overrideNote} onChange={(e) => setOverrideNote(e.target.value)} />
                </div>
                <button className="btn">Apply override</button>
              </form>
            </div>
          )}

          {!isEvaluator && script && (
            <div className="card stack">
              <h2>Identity</h2>
              <div><span className="hint">Student</span><div>{script.student_name}</div></div>
              <div><span className="hint">Matriculation</span><div className="mono">{script.student_no}</div></div>
              <div><span className="hint">Session</span><div>{script.session}</div></div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

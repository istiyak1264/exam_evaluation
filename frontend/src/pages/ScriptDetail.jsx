import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, money, when } from '../api'
import { useAuth } from '../auth'
import { PdfPanel } from '../components/PdfPanel'
import { Badge, Grade, Loading, Note } from '../components/ui'
import { Breadcrumbs, Caption, SummaryList, TaskList } from '../components/gov'

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
      <Breadcrumbs items={[
        { label: isEvaluator ? 'My marking' : 'Scripts', href: isEvaluator ? '/queue' : '/scripts' },
        { label: blind?.script_code ?? script?.script_code ?? id },
      ]} />

      <div className="page-head">
        <Caption>
          {blind?.course_code ?? script?.course_code} · {blind?.course_title ?? script?.course_title} · paper out of {maxMarks}
          {!isEvaluator && script ? ` · ${script.page_count ?? '?'} page(s)` : ''}
        </Caption>
        <h1 className="mono">{blind?.script_code ?? script?.script_code}</h1>
        <div className="row" style={{ marginTop: '.6rem' }}>
          <Badge status={status} />
          {final !== null && <strong className="mono">Final {money(final)}</strong>}
          {final !== null && <Grade grade={final >= 70 ? 'A' : final >= 60 ? 'B' : final >= 50 ? 'C' : final >= 45 ? 'D' : final >= 40 ? 'E' : 'F'} />}
        </div>
      </div>

      {msg && <Note kind="info">{msg}</Note>}

      {isEvaluator && (
        <Note kind="info">
          Blinded view — no student name or matriculation number is shown. Your mark locks once submitted.
        </Note>
      )}

      <section className="card">
        <h2>Marking progress</h2>
        <TaskList
          items={[
            { label: 'Script registered and anonymised', status: 'done', meta: 'Completed' },
            { label: 'Two examiners appointed', status: status === 'REGISTERED' ? 'todo' : 'done', meta: status === 'REGISTERED' ? 'Not started' : 'Completed' },
            { label: 'First and second marks submitted', status: marks.length >= 2 ? 'done' : status === 'EVAL_PENDING' ? 'current' : 'todo' },
            { label: 'Third examination (gap of 13 or more)', hint: status === 'THIRD_REVIEW' ? 'A third examiner is required before this script can be resolved' : 'Only needed when the first two marks differ by 13 or more', status: status === 'THIRD_REVIEW' ? 'current' : final !== null ? 'done' : 'todo', meta: status === 'THIRD_REVIEW' ? 'Required' : final !== null && script?.method === 'MEDIAN_OF_THREE' ? 'Completed' : 'Not needed' },
            { label: 'Result resolved', status: final !== null ? 'done' : 'todo', meta: final !== null ? 'Completed' : 'Not started' },
          ]}
        />
      </section>

      <div className="detail-grid">
        <div className="card">
          <h2>Script pages</h2>
          <PdfPanel scriptId={id} />
        </div>

        <div className="stack">
          <div className="card">
            <h2>Marks</h2>
            {marks.length === 0 ? (
              <p className="hint">Nothing submitted yet.</p>
            ) : (
              <SummaryList
                rows={[
                  ...marks.map((m) => ({
                    key: `${m.slot.toLowerCase()} examiner${!isEvaluator && m.evaluator_name ? ` · ${m.evaluator_name}` : ''}`,
                    value: (
                      <>
                        <strong className="mono">{money(m.score)}</strong>
                        {m.comment ? <div className="hint">{m.comment}</div> : null}
                        <div className="hint">{when(m.submitted_at)}</div>
                      </>
                    ),
                  })),
                  ...(final !== null
                    ? [{
                        key: 'Final marks',
                        value: (
                          <>
                            <strong className="mono">{money(final)}</strong>
                            <div className="hint">{(script?.method ?? blind?.method)?.replace(/_/g, ' ').toLowerCase() ?? 'board decision'}</div>
                          </>
                        ),
                      }]
                    : []),
                ]}
              />
            )}
            {status === 'THIRD_REVIEW' && (
              <div className="note" style={{ marginTop: '1rem' }}>
                A gap of 13 or more was recorded. A third external examiner must mark this script.
              </div>
            )}
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
            <div className="card">
              <h2>Identity</h2>
              <SummaryList
                rows={[
                  { key: 'Student', value: script.student_name },
                  { key: 'Matriculation', value: <span className="mono">{script.student_no}</span> },
                  { key: 'Session', value: script.session },
                ]}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

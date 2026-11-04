import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import { Note } from '../components/ui'

export default function RegisterScript() {
  const [sessions, setSessions] = useState([])
  const [sessionId, setSessionId] = useState('')
  const [courses, setCourses] = useState([])
  const [courseId, setCourseId] = useState('')
  const [term, setTerm] = useState('')
  const [students, setStudents] = useState([])
  const [studentId, setStudentId] = useState('')
  const [code, setCode] = useState('')
  const [file, setFile] = useState(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const [done, setDone] = useState(null)

  useEffect(() => {
    api.sessions().then((r) => {
      setSessions(r.sessions)
      const active = r.sessions.find((s) => s.is_active)
      if (active) setSessionId(active.id)
    }).catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    if (!sessionId) return
    api.courses(sessionId).then((r) => setCourses(r.courses)).catch(() => setCourses([]))
  }, [sessionId])

  useEffect(() => {
    const t = setTimeout(() => {
      api.students(term, sessionId).then((r) => setStudents(r.students.slice(0, 25))).catch(() => setStudents([]))
    }, 220)
    return () => clearTimeout(t)
  }, [term, sessionId])

  async function submit(e) {
    e.preventDefault()
    setError(null)
    if (!courseId || !studentId || !file) {
      setError('Pick a course, a student and the scanned PDF.')
      return
    }
    const form = new FormData()
    form.append('course_id', courseId)
    form.append('student_id', studentId)
    if (code.trim()) form.append('script_code', code.trim())
    form.append('file', file)
    setBusy(true)
    try {
      const res = await api.registerScript(form)
      setDone({ id: res.script.id, code: res.script_code })
      setFile(null)
      setCode('')
      setStudentId('')
    } catch (err) {
      setError((err).message)
    } finally {
      setBusy(false)
    }
  }

  const student = students.find((s) => s.id === studentId)

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Register a script</h1>
          <p>
            The PDF must already have the student's name and number removed. The office records the link between the
            cover-sheet code and the student here; examiners only ever see the code.
          </p>
        </div>
        <Link className="btn" to="/scripts">Back to scripts</Link>
      </div>

      {done && (
        <Note kind="info">
          Registered. Cover-sheet code <strong className="mono">{done.code}</strong> — write this on the script bundle.{' '}
          <Link to={`/scripts/${done.id}`}>Open the script</Link>
        </Note>
      )}
      {error && <Note kind="danger">{error}</Note>}

      <form className="card stack" onSubmit={submit}>
        <div className="row">
          <div className="field" style={{ flex: 1 }}>
            <label>Academic session</label>
            <select value={sessionId} onChange={(e) => { setSessionId(e.target.value); setCourseId(''); setStudentId('') }}>
              {sessions.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          </div>
          <div className="field" style={{ flex: 2 }}>
            <label>Course</label>
            <select value={courseId} onChange={(e) => setCourseId(e.target.value)}>
              <option value="">Select a course…</option>
              {courses.map((c) => <option key={c.id} value={c.id}>{c.code} — {c.title}</option>)}
            </select>
          </div>
        </div>

        <div className="field">
          <label>Student (search by name or matriculation number)</label>
          <input value={term} onChange={(e) => setTerm(e.target.value)} placeholder="e.g. Ada Lovelace or MAT/2026/041" />
          <select value={studentId} onChange={(e) => setStudentId(e.target.value)} size={5}>
            {students.map((s) => (
              <option key={s.id} value={s.id}>{s.full_name} — {s.student_no}{s.session ? ` (${s.session})` : ''}</option>
            ))}
          </select>
          {student && <span className="hint">Selected: {student.full_name} · {student.student_no}</span>}
        </div>

        <div className="field-row">
          <div className="field">
            <label htmlFor="pdf">PDF scan</label>
            <input id="pdf" type="file" accept="application/pdf" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
            <span className="hint">Max 25 MB. Header is validated as %PDF-.</span>
          </div>
          <div className="field">
            <label htmlFor="code">Cover-sheet code (optional)</label>
            <input id="code" value={code} onChange={(e) => setCode(e.target.value)} placeholder="auto-generated if left blank" />
            <span className="hint">Printed on the cover sheet; leave blank to auto-generate.</span>
          </div>
        </div>

        <div className="row">
          <button className="btn primary" disabled={busy}>{busy ? 'Uploading…' : 'Register script'}</button>
          <span className="hint">One script per student per course.</span>
        </div>
      </form>
    </div>
  )
}

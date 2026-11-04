import { useState } from 'react'
import { api } from '../api'
import { useAuth } from '../auth'
import { Empty, Loading, Note, useFetch } from '../components/ui'

export default function Catalog() {
  const { user } = useAuth()
  const isAdmin = user?.role === 'ADMIN'
  const sessions = useFetch(() => api.sessions(), [])
  const [sessionId, setSessionId] = useState('')
  const courses = useFetch(() => api.courses(sessionId), [sessionId])
  const students = useFetch(() => api.students('', sessionId), [sessionId])
  const [msg, setMsg] = useState<string | null>(null)
  const [sesForm, setSesForm] = useState({ name: '', code: '' })
  const [crsForm, setCrsForm] = useState({ code: '', title: '', max_marks: '100', pass_marks: '40' })
  const [stuForm, setStuForm] = useState({ student_no: '', full_name: '', email: '' })

  async function run(fn: () => Promise<unknown>, note: string) {
    setMsg(null)
    try { await fn(); setMsg(note); sessions.reload(); courses.reload(); students.reload() } catch (e) { setMsg((e as Error).message) }
  }

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Sessions, courses & students</h1>
          <p>Master data. Scripts can only be registered against an existing course and student.</p>
        </div>
        <select value={sessionId} onChange={(e) => setSessionId(e.target.value)} style={{ padding: '.5rem', borderRadius: 8, border: '1px solid var(--line)' }}>
          <option value="">All sessions</option>
          {sessions.data?.sessions.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
        </select>
      </div>

      {msg && <Note kind="info">{msg}</Note>}

      {isAdmin && (
        <div className="card stack">
          <h2>New academic session</h2>
          <form className="row" onSubmit={(e) => { e.preventDefault(); void run(() => api.createSession(sesForm.name, sesForm.code), 'Session created.') }}>
            <input required placeholder="2026/2027" value={sesForm.name} onChange={(e) => setSesForm({ ...sesForm, name: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
            <input required placeholder="code e.g. A26" value={sesForm.code} onChange={(e) => setSesForm({ ...sesForm, code: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
            <button className="btn primary">Add session</button>
          </form>
          <table>
            <thead><tr><th>Session</th><th>Active</th><th /></tr></thead>
            <tbody>
              {sessions.data?.sessions.map((s) => (
                <tr key={s.id}>
                  <td>{s.name} <span className="hint mono">{s.code}</span></td>
                  <td>{s.is_active ? 'yes' : 'no'}</td>
                  <td><button className="btn sm" onClick={() => void run(() => api.setSessionActive(s.id, !s.is_active), 'Session updated.')}>{s.is_active ? 'Close' : 'Reopen'}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="card stack">
        <h2>Courses</h2>
        <form className="row" onSubmit={(e) => {
          e.preventDefault()
          if (!sessionId) { setMsg('Pick a session first.'); return }
          void run(() => api.createCourse({ ...crsForm, session_id: sessionId, max_marks: Number(crsForm.max_marks), pass_marks: Number(crsForm.pass_marks) }), 'Course created.')
        }}>
          <input required placeholder="CSC301" value={crsForm.code} onChange={(e) => setCrsForm({ ...crsForm, code: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
          <input required placeholder="Title" value={crsForm.title} onChange={(e) => setCrsForm({ ...crsForm, title: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8, flex: 1 }} />
          <input required type="number" placeholder="max" value={crsForm.max_marks} onChange={(e) => setCrsForm({ ...crsForm, max_marks: e.target.value })} style={{ width: 80, padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
          <input required type="number" placeholder="pass" value={crsForm.pass_marks} onChange={(e) => setCrsForm({ ...crsForm, pass_marks: e.target.value })} style={{ width: 80, padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
          <button className="btn primary">Add course</button>
        </form>
        {courses.loading && <Loading />}
        {courses.data?.courses.length === 0 && <Empty>No courses.</Empty>}
        {courses.data && courses.data.courses.length > 0 && (
          <table>
            <thead><tr><th>Code</th><th>Title</th><th>Session</th><th>Max / pass</th><th>Scripts</th></tr></thead>
            <tbody>
              {courses.data.courses.map((c) => (
                <tr key={c.id}><td className="mono">{c.code}</td><td>{c.title}</td><td className="hint">{c.session}</td><td className="mono">{c.max_marks} / {c.pass_marks}</td><td>{c.script_count}</td></tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="card stack">
        <h2>Students</h2>
        <form className="row" onSubmit={(e) => {
          e.preventDefault()
          if (!sessionId) { setMsg('Pick a session first.'); return }
          void run(() => api.createStudent({ ...stuForm, session_id: sessionId }), 'Student added.')
        }}>
          <input required placeholder="Matriculation no" value={stuForm.student_no} onChange={(e) => setStuForm({ ...stuForm, student_no: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
          <input required placeholder="Full name" value={stuForm.full_name} onChange={(e) => setStuForm({ ...stuForm, full_name: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8, flex: 1 }} />
          <input placeholder="Email (optional)" value={stuForm.email} onChange={(e) => setStuForm({ ...stuForm, email: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
          <button className="btn primary">Add student</button>
        </form>
        {students.data?.students.length === 0 && <Empty>No students.</Empty>}
        {students.data && students.data.students.length > 0 && (
          <table>
            <thead><tr><th>Matriculation</th><th>Name</th><th>Session</th><th>Portal account</th></tr></thead>
            <tbody>
              {students.data.students.map((s) => (
                <tr key={s.id}><td className="mono">{s.student_no}</td><td>{s.full_name}</td><td className="hint">{s.session}</td><td>{s.user_id ? <span className="badge ok">linked</span> : <span className="badge">none</span>}</td></tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

import { useEffect, useMemo, useRef, useState } from 'react'
import { api } from '../api'
import { useAuth } from '../auth'
import { Empty, Loading, Note, useFetch } from '../components/ui'

// CoursePicker is a syllabus dropdown: terms list their courses, and the two
// optional slots stay collapsed behind an arrow until clicked.
function CoursePicker({ courses, value, onChange }) {
  const [open, setOpen] = useState(false)
  const [expanded, setExpanded] = useState({})
  const box = useRef(null)

  useEffect(() => {
    function away(e) {
      if (box.current && !box.current.contains(e.target)) setOpen(false)
    }
    document.addEventListener('mousedown', away)
    return () => document.removeEventListener('mousedown', away)
  }, [])

  const terms = useMemo(() => {
    const out = new Map()
    for (const c of courses) {
      if (!out.has(c.term)) out.set(c.term, [])
      out.get(c.term).push(c)
    }
    return [...out.entries()]
  }, [courses])

  const chosen = courses.find((c) => c.id === value)
  const toggle = (key) => setExpanded((x) => ({ ...x, [key]: !x[key] }))

  function pick(id) {
    onChange(id)
    setOpen(false)
  }

  function options(list) {
    return list.map((c) => (
      <button key={c.id} type="button" className={`picker-opt ${c.id === value ? 'is-on' : ''}`} onClick={() => pick(c.id)}>
        <span className="mono">{c.code}</span> — {c.title}
        <span className="picker-credit">{c.credit}</span>
      </button>
    ))
  }

  function optional(term, label, category) {
    const list = courses.filter((c) => c.term === term && c.category === category)
    if (list.length === 0) return null
    const key = `${term}|${category}`
    const isOpen = !!expanded[key]
    return (
      <>
        <button type="button" className="picker-toggle" onClick={() => toggle(key)} aria-expanded={isOpen}>
          <span>{label}</span>
          <span className="picker-count">{list.length}</span>
          <span className={`picker-arrow ${isOpen ? 'up' : ''}`} aria-hidden="true">▾</span>
        </button>
        {isOpen && <div className="picker-nest">{options(list)}</div>}
      </>
    )
  }

  return (
    <div className="picker" ref={box}>
      <button type="button" className="picker-trigger" onClick={() => setOpen((o) => !o)} aria-haspopup="listbox" aria-expanded={open}>
        {chosen ? (
          <span><span className="mono">{chosen.code}</span> — {chosen.title}</span>
        ) : (
          <span className="picker-placeholder">Select a course from the curriculum…</span>
        )}
        <span className="picker-arrow" aria-hidden="true">▾</span>
      </button>
      {open && (
        <div className="picker-panel" role="listbox">
          {terms.map(([term, list]) => (
            <div key={term} className="picker-term">
              <div className="picker-term-head">{term}</div>
              {options(list.filter((c) => c.category === 'CORE'))}
              {optional(term, 'OPTIONAL-I', 'OPTIONAL_I')}
              {optional(term, 'OPTIONAL-II', 'OPTIONAL_II')}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default function Catalog() {
  const { user } = useAuth()
  const isAdmin = user?.role === 'ADMIN'
  const sessions = useFetch(() => api.sessions(), [])
  const [sessionId, setSessionId] = useState('')
  const courses = useFetch(() => api.courses(sessionId), [sessionId])
  const students = useFetch(() => api.students('', sessionId), [sessionId])
  const portalUsers = useFetch(() => api.users('STUDENT'), [])
  const catalog = useFetch(() => api.courseCatalog(), [])
  const [msg, setMsg] = useState(null)
  const [sesName, setSesName] = useState('')
  const [catalogId, setCatalogId] = useState('')
  const [stuForm, setStuForm] = useState({ student_no: '', registration_no: '', full_name: '', email: '' })

  const picked = catalog.data?.courses.find((c) => c.id === catalogId)

  // Group the fixed curriculum by study term so the picker reads like the syllabus.
  const terms = useMemo(() => {
    const out = new Map()
    for (const c of catalog.data?.courses ?? []) {
      if (!out.has(c.term)) out.set(c.term, [])
      out.get(c.term).push(c)
    }
    return [...out.entries()]
  }, [catalog.data])

  async function run(fn, note) {
    setMsg(null)
    try {
      await fn()
      setMsg(note)
      sessions.reload()
      courses.reload()
      students.reload()
    } catch (e) {
      setMsg(e.message)
    }
  }

  function removeSession(s) {
    if (!window.confirm(`Delete session ${s.name}? Its courses, students and scripts go with it.`)) return
    void run(() => api.deleteSession(s.id), `Session ${s.name} deleted.`)
  }

  function removeStudent(s) {
    if (!window.confirm(`Delete student ${s.full_name}? Their registered scripts go too.`)) return
    void run(() => api.deleteStudent(s.id), `${s.full_name} removed from the session.`)
  }

  function removeCourse(c) {
    const warn = c.script_count > 0 ? ` Its ${c.script_count} script(s), marks and results go with it.` : ''
    if (!window.confirm(`Delete course ${c.code} — ${c.title}?${warn}`)) return
    void run(() => api.deleteCourse(c.id), `${c.code} removed from the session.`)
  }

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Sessions, courses & students</h1>
          <p>Master data. A script can only be registered against an existing course and student.</p>
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
          <form className="row" onSubmit={(e) => {
            e.preventDefault()
            void run(async () => { await api.createSession(sesName); setSesName('') }, 'Session created.')
          }}>
            <input required placeholder="2026/2027" value={sesName} onChange={(e) => setSesName(e.target.value)} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8, flex: 1 }} />
            <button className="btn primary">Add session</button>
          </form>
          <table>
            <thead><tr><th>Session</th><th>Active</th><th /></tr></thead>
            <tbody>
              {sessions.data?.sessions.map((s) => (
                <tr key={s.id}>
                  <td>{s.name}</td>
                  <td>{s.is_active ? 'yes' : 'no'}</td>
                  <td><button className="btn sm danger" onClick={() => removeSession(s)}>Delete</button></td>
                </tr>
              ))}
            </tbody>
          </table>
          {sessions.data?.sessions.length === 0 && <Empty>No sessions yet.</Empty>}
        </div>
      )}

      <div className="card stack">
        <h2>Courses</h2>
        <form className="row" onSubmit={(e) => {
          e.preventDefault()
          if (!sessionId) { setMsg('Pick a session first.'); return }
          if (!catalogId) { setMsg('Pick a course from the curriculum.'); return }
          void run(async () => { await api.createCourse({ catalog_id: catalogId, session_id: sessionId }); setCatalogId('') }, 'Course added.')
        }}>
          <CoursePicker courses={catalog.data?.courses ?? []} value={catalogId} onChange={setCatalogId} />
          {picked && <span className="badge info">credit {picked.credit}</span>}
          <button className="btn primary" disabled={!picked}>Add course</button>
        </form>

        {courses.loading && <Loading />}
        {courses.data?.courses.length === 0 && <Empty>No courses in this view.</Empty>}
        {courses.data && courses.data.courses.length > 0 && (
          <table>
            <thead><tr><th>Code</th><th>Title</th><th>Credit</th><th>Scripts</th><th /></tr></thead>
            <tbody>
              {courses.data.courses.map((c) => (
                <tr key={c.id}>
                  <td className="mono">{c.code}</td>
                  <td>{c.title}</td>
                  <td className="mono">{c.credit}</td>
                  <td>{c.script_count}</td>
                  <td><button className="btn sm danger" onClick={() => removeCourse(c)}>Delete</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        <details>
          <summary className="hint">Full curriculum ({catalog.data?.courses.length ?? 0} courses)</summary>
          {terms.map(([term, list]) => (
            <div key={term} style={{ marginTop: '.8rem' }}>
              <strong>{term}</strong>
              <table>
                <thead><tr><th>Code</th><th>Title</th><th>Credit</th></tr></thead>
                <tbody>
                  {list.map((c) => (
                    <tr key={c.id}>
                      <td className="mono">{c.code}</td>
                      <td>{c.title}{c.category !== 'CORE' && <span className="hint"> · optional</span>}</td>
                      <td className="mono">{c.credit}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </details>
      </div>

      <div className="card stack">
        <h2>Students</h2>
        <form className="row" onSubmit={(e) => {
          e.preventDefault()
          if (!sessionId) { setMsg('Pick a session first.'); return }
          void run(() => api.createStudent({ ...stuForm, session_id: sessionId }), 'Student added.')
        }}>
          <input required placeholder="Student ID" value={stuForm.student_no} onChange={(e) => setStuForm({ ...stuForm, student_no: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
          <input required placeholder="Registration no" value={stuForm.registration_no} onChange={(e) => setStuForm({ ...stuForm, registration_no: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
          <input required placeholder="Full name" value={stuForm.full_name} onChange={(e) => setStuForm({ ...stuForm, full_name: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8, flex: 1 }} />
          <input placeholder="Email (optional)" value={stuForm.email} onChange={(e) => setStuForm({ ...stuForm, email: e.target.value })} style={{ padding: '.5rem', border: '1px solid var(--line)', borderRadius: 8 }} />
          <button className="btn primary">Add student</button>
        </form>
        {students.data?.students.length === 0 && <Empty>No students.</Empty>}
        {students.data && students.data.students.length > 0 && (
          <table>
            <thead><tr><th>Student ID</th><th>Registration</th><th>Name</th><th>Session</th><th>Portal account</th><th /></tr></thead>
            <tbody>
              {students.data.students.map((s) => (
                <tr key={s.id}>
                  <td className="mono">{s.student_no}</td><td className="mono">{s.registration_no ?? '—'}</td><td>{s.full_name}</td><td className="hint">{s.session}</td>
                  <td>
                    {s.user_id ? <span className="badge ok">linked</span> : (
                      <select
                        defaultValue=""
                        onChange={(e) => { if (e.target.value) void run(() => api.linkStudentUser(s.id, e.target.value), 'Portal account linked.') }}
                        style={{ padding: '.35rem', border: '1px solid var(--line)', borderRadius: 6 }}
                      >
                        <option value="">link account…</option>
                        {(portalUsers.data?.users ?? []).map((u) => (
                          <option key={u.id} value={u.id}>{u.email}</option>
                        ))}
                      </select>
                    )}
                  </td>
                  <td><button className="btn sm danger" onClick={() => removeStudent(s)}>Delete</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

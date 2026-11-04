// Thin API client: attaches the access token, refreshes once on 401.

export type Role = 'ADMIN' | 'HALL_MANAGER' | 'EVALUATOR' | 'STUDENT'

export interface User {
  id: string
  email: string
  full_name: string
  role: Role
  is_active: boolean
  created_at: string
}

export interface Session { id: string; name: string; code: string; is_active: boolean }
export interface Student {
  id: string; student_no: string; full_name: string; email?: string | null
  session_id: string; session?: string; user_id?: string | null
}
export interface Course {
  id: string; code: string; title: string; session_id: string; session?: string
  max_marks: number; pass_marks: number; instructor_name?: string | null; script_count: number
}
export interface Mark {
  slot: 'FIRST' | 'SECOND' | 'THIRD'
  score: number
  comment?: string | null
  evaluator_name?: string
  submitted_at: string
}
export interface Script {
  id: string; script_code: string; status: string; course_id: string
  course_code: string; course_title: string; max_marks: number
  student_id: string; student_no: string; student_name: string; session: string
  page_count?: number | null; pdf_bytes: number; marks: Mark[]
  final_marks?: number | null; gap?: number | null; method?: string | null
  registered_at: string; resolved_at?: string | null
}
export interface BlindScript {
  id: string; script_code: string; status: string; course_code: string
  course_title: string; max_marks: number; page_count?: number | null
  my_slot: string; marks: Mark[]; final_marks?: number | null; assigned_at: string
}
export interface ResultRow {
  script_id: string; script_code: string; student_id: string; student_no: string
  student_name: string; session: string; course_code: string; course_title: string
  status: string; final_marks?: number | null; grade?: string | null
  passed?: boolean | null; challenge_status?: string | null
  resolved_at?: string | null; registered_at: string
}
export interface Challenge {
  id: string; script_id: string; script_code: string; student_no?: string
  student_name?: string; course_code?: string; reason: string; status: string
  decision_note?: string | null; created_at: string; decided_at?: string | null
}
export interface AuditEntry {
  id: number; actor_name?: string | null; actor_role?: string | null
  action: string; entity: string; entity_id?: string | null; meta: unknown; created_at: string
}
export interface Stats {
  scripts_by_status: Record<string, number>
  users_by_role: Record<string, number>
  open_challenges: number; results_resolved: number
  third_review_rate: number; average_final_marks: number
}

const ACCESS = 'see.access'
const REFRESH = 'see.refresh'

export const tokens = {
  get access() { return localStorage.getItem(ACCESS) },
  get refresh() { return localStorage.getItem(REFRESH) },
  set(access: string, refresh: string) {
    localStorage.setItem(ACCESS, access)
    localStorage.setItem(REFRESH, refresh)
  },
  clear() { localStorage.removeItem(ACCESS); localStorage.removeItem(REFRESH) },
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

let refreshing: Promise<boolean> | null = null

async function refreshOnce(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      const rt = tokens.refresh
      if (!rt) return false
      const res = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: rt }),
      })
      if (!res.ok) { tokens.clear(); return false }
      const body = await res.json()
      tokens.set(body.access_token, body.refresh_token)
      return true
    })().finally(() => { setTimeout(() => { refreshing = null }, 50) })
  }
  return refreshing
}

async function request<T>(method: string, path: string, body?: unknown, retry = true): Promise<T> {
  const headers: Record<string, string> = {}
  const access = tokens.access
  if (access) headers.Authorization = `Bearer ${access}`
  let payload: BodyInit | undefined
  if (body instanceof FormData) {
    payload = body
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  const res = await fetch(path, { method, headers, body: payload })
  if (res.status === 401 && retry && tokens.refresh) {
    if (await refreshOnce()) return request<T>(method, path, body, false)
    tokens.clear()
  }
  if (!res.ok) {
    let msg = res.statusText
    try {
      const b = await res.json()
      if (b?.error) msg = b.error
    } catch { /* non-JSON error */ }
    throw new ApiError(res.status, msg)
  }
  if (res.status === 204) return undefined as T
  const ct = res.headers.get('Content-Type') ?? ''
  if (!ct.includes('application/json')) return (await res.text()) as unknown as T
  return (await res.json()) as T
}

export const api = {
  login: (email: string, password: string) =>
    request<{ access_token: string; refresh_token: string; user: User }>('POST', '/api/auth/login', { email, password }),
  logout: () => request('POST', '/api/auth/logout', { refresh_token: tokens.refresh }),
  me: () => request<{ user: User; student?: Student }>('GET', '/api/auth/me'),
  changePassword: (current_password: string, new_password: string) =>
    request('POST', '/api/auth/password', { current_password, new_password }),

  sessions: () => request<{ sessions: Session[] }>('GET', '/api/sessions'),
  createSession: (name: string, code: string) => request<Session>('POST', '/api/sessions', { name, code }),
  setSessionActive: (id: string, is_active: boolean) => request('PATCH', `/api/sessions/${id}`, { is_active }),

  students: (q = '', session_id = '') =>
    request<{ students: Student[] }>('GET', `/api/students?q=${encodeURIComponent(q)}&session_id=${encodeURIComponent(session_id)}`),
  createStudent: (s: { student_no: string; full_name: string; email?: string; session_id: string }) =>
    request<Student>('POST', '/api/students', s),
  linkStudentUser: (id: string, user_id: string) => request('POST', `/api/students/${id}/link-user`, { user_id }),

  courses: (session_id = '') => request<{ courses: Course[] }>('GET', `/api/courses?session_id=${encodeURIComponent(session_id)}`),
  createCourse: (c: { code: string; title: string; session_id: string; max_marks: number; pass_marks: number }) =>
    request<Course>('POST', '/api/courses', c),

  users: (role = '', q = '') => request<{ users: User[] }>('GET', `/api/users?role=${role}&q=${encodeURIComponent(q)}`),
  createUser: (u: { email: string; full_name: string; role: Role; password: string }) => request<User>('POST', '/api/users', u),
  patchUser: (id: string, p: { is_active?: boolean; password?: string }) => request('PATCH', `/api/users/${id}`, p),

  scripts: (params: { status?: string; course_id?: string; q?: string } = {}) => {
    const qs = new URLSearchParams(Object.entries(params).filter(([, v]) => v) as [string, string][])
    return request<{ scripts: Script[] }>('GET', `/api/scripts?${qs.toString()}`)
  },
  script: (id: string) => request<Script | BlindScript>('GET', `/api/scripts/${id}`),
  registerScript: (form: FormData) => request<{ script: Script; script_code: string }>('POST', '/api/scripts', form),
  assign: (id: string, first: string, second: string) =>
    request('POST', `/api/scripts/${id}/assign`, { first_evaluator_id: first, second_evaluator_id: second }),
  assignThird: (id: string, evaluator_id: string) => request('POST', `/api/scripts/${id}/assign-third`, { evaluator_id }),
  pdfToken: (id: string) => request<{ url: string }>('POST', `/api/scripts/${id}/pdf-token`),
  mark: (id: string, score: number, comment: string) =>
    request<{ status: string; final_marks?: number; method?: string; gap?: number }>('POST', `/api/scripts/${id}/marks`, { score, comment }),
  queue: (status = '') => request<{ queue: BlindScript[] }>('GET', `/api/queue?status=${status}`),
  override: (id: string, score: number, note: string) => request('POST', `/api/scripts/${id}/override`, { score, note }),

  results: (params: Record<string, string>) => {
    const qs = new URLSearchParams(Object.entries(params).filter(([, v]) => v))
    return request<{ results: ResultRow[]; count: number }>('GET', `/api/results?${qs.toString()}`)
  },
  myResults: () => request<{ results: ResultRow[]; student: Student }>('GET', '/api/results/mine'),

  challenges: (status = '') => request<{ challenges: Challenge[] }>('GET', `/api/challenges?status=${status}`),
  createChallenge: (script_id: string, reason: string) => request<Challenge>('POST', '/api/challenges', { script_id, reason }),
  decideChallenge: (id: string, status: 'UPHELD' | 'DISMISSED', note: string) =>
    request('PATCH', `/api/challenges/${id}`, { status, note }),

  stats: () => request<Stats>('GET', '/api/admin/stats'),
  audit: (entity = '', entity_id = '') =>
    request<{ entries: AuditEntry[] }>('GET', `/api/admin/audit?entity=${entity}&entity_id=${entity_id}`),
}

export function money(n?: number | null) {
  return n === null || n === undefined ? '—' : n.toFixed(2)
}

export function when(iso?: string | null) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

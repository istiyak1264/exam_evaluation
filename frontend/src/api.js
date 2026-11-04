// Thin API client: attaches the access token and refreshes once on 401.

const ACCESS = 'see.access'
const REFRESH = 'see.refresh'

export const tokens = {
  get access() {
    return localStorage.getItem(ACCESS)
  },
  get refresh() {
    return localStorage.getItem(REFRESH)
  },
  set(access, refresh) {
    localStorage.setItem(ACCESS, access)
    localStorage.setItem(REFRESH, refresh)
  },
  clear() {
    localStorage.removeItem(ACCESS)
    localStorage.removeItem(REFRESH)
  },
}

export class ApiError extends Error {
  constructor(status, message) {
    super(message)
    this.status = status
  }
}

let refreshing = null

async function refreshOnce() {
  if (!refreshing) {
    refreshing = (async () => {
      const rt = tokens.refresh
      if (!rt) return false
      const res = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: rt }),
      })
      if (!res.ok) {
        tokens.clear()
        return false
      }
      const body = await res.json()
      tokens.set(body.access_token, body.refresh_token)
      return true
    })().finally(() => {
      setTimeout(() => {
        refreshing = null
      }, 50)
    })
  }
  return refreshing
}

async function request(method, path, body, retry = true) {
  const headers = {}
  const access = tokens.access
  if (access) headers.Authorization = `Bearer ${access}`
  let payload
  if (body instanceof FormData) {
    payload = body
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  const res = await fetch(path, { method, headers, body: payload })
  if (res.status === 401 && retry && tokens.refresh) {
    if (await refreshOnce()) return request(method, path, body, false)
    tokens.clear()
  }
  if (!res.ok) {
    let msg = res.statusText
    try {
      const b = await res.json()
      if (b?.error) msg = b.error
    } catch {
      /* non-JSON error */
    }
    throw new ApiError(res.status, msg)
  }
  if (res.status === 204) return undefined
  const ct = res.headers.get('Content-Type') ?? ''
  if (!ct.includes('application/json')) return await res.text()
  return await res.json()
}

const query = (params) =>
  new URLSearchParams(Object.entries(params).filter(([, v]) => v !== '' && v !== undefined && v !== null)).toString()

export const api = {
  login: (email, password) => request('POST', '/api/auth/login', { email, password }),
  logout: () => request('POST', '/api/auth/logout', { refresh_token: tokens.refresh }),
  register: (payload) => request('POST', '/api/auth/register', payload),
  me: () => request('GET', '/api/auth/me'),
  changePassword: (current_password, new_password) =>
    request('POST', '/api/auth/password', { current_password, new_password }),

  sessions: () => request('GET', '/api/sessions'),
  createSession: (name, code) => request('POST', '/api/sessions', { name, code }),
  setSessionActive: (id, is_active) => request('PATCH', `/api/sessions/${id}`, { is_active }),

  students: (q = '', session_id = '') =>
    request('GET', `/api/students?${query({ q, session_id })}`),
  createStudent: (student) => request('POST', '/api/students', student),
  linkStudentUser: (id, user_id) => request('POST', `/api/students/${id}/link-user`, { user_id }),

  courses: (session_id = '') => request('GET', `/api/courses?${query({ session_id })}`),
  createCourse: (course) => request('POST', '/api/courses', course),

  users: (role = '', q = '') => request('GET', `/api/users?${query({ role, q })}`),
  createUser: (user) => request('POST', '/api/users', user),
  patchUser: (id, patch) => request('PATCH', `/api/users/${id}`, patch),
  deleteUser: (id) => request('DELETE', `/api/users/${id}`),

  scripts: (params = {}) => request('GET', `/api/scripts?${query(params)}`),
  script: (id) => request('GET', `/api/scripts/${id}`),
  registerScript: (form) => request('POST', '/api/scripts', form),
  assign: (id, first, second) =>
    request('POST', `/api/scripts/${id}/assign`, { first_evaluator_id: first, second_evaluator_id: second }),
  assignThird: (id, evaluator_id) => request('POST', `/api/scripts/${id}/assign-third`, { evaluator_id }),
  pdfToken: (id) => request('POST', `/api/scripts/${id}/pdf-token`),
  mark: (id, score, comment) => request('POST', `/api/scripts/${id}/marks`, { score, comment }),
  queue: (status = '') => request('GET', `/api/queue?${query({ status })}`),
  override: (id, score, note) => request('POST', `/api/scripts/${id}/override`, { score, note }),

  results: (params) => request('GET', `/api/results?${query(params)}`),
  myResults: () => request('GET', '/api/results/mine'),

  challenges: (status = '') => request('GET', `/api/challenges?${query({ status })}`),
  createChallenge: (script_id, reason) => request('POST', '/api/challenges', { script_id, reason }),
  decideChallenge: (id, status, note) => request('PATCH', `/api/challenges/${id}`, { status, note }),

  stats: () => request('GET', '/api/admin/stats'),
  audit: (entity = '', entity_id = '') => request('GET', `/api/admin/audit?${query({ entity, entity_id })}`),
}

export function money(n) {
  return n === null || n === undefined ? '—' : n.toFixed(2)
}

export function when(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

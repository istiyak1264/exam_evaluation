import { useEffect, useState } from 'react'
import { Navigate, NavLink, Route, Routes, useLocation } from 'react-router-dom'
import { useAuth } from './auth'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Scripts from './pages/Scripts'
import RegisterScript from './pages/RegisterScript'
import ScriptDetail from './pages/ScriptDetail'
import Queue from './pages/Queue'
import Search from './pages/Search'
import MyResults from './pages/MyResults'
import Challenges from './pages/Challenges'
import Users from './pages/Users'
import Catalog from './pages/Catalog'
import Audit from './pages/Audit'
import Account from './pages/Account'

const home = {
  ADMIN: '/dashboard',
  HALL_MANAGER: '/scripts',
  EVALUATOR: '/queue',
  STUDENT: '/me',
}

function Guard({ roles, children }) {
  const { user, loading } = useAuth()
  const loc = useLocation()
  if (loading) return <div className="main">Loading…</div>
  if (!user) return <Navigate to="/login" state={{ from: loc.pathname }} replace />
  if (!roles.includes(user.role)) return <Navigate to={home[user.role]} replace />
  return children
}

const links = [
  { to: '/dashboard', label: 'Dashboard', roles: ['ADMIN'] },
  { to: '/scripts', label: 'Scripts', roles: ['ADMIN', 'HALL_MANAGER'] },
  { to: '/scripts/new', label: 'Register script', roles: ['ADMIN', 'HALL_MANAGER'] },
  { to: '/search', label: 'Find results', roles: ['ADMIN', 'HALL_MANAGER', 'EVALUATOR'] },
  { to: '/challenges', label: 'Challenges', roles: ['ADMIN', 'HALL_MANAGER'] },
  { to: '/queue', label: 'My marking', roles: ['EVALUATOR'] },
  { to: '/me', label: 'My results', roles: ['STUDENT'] },
  { to: '/admin/users', label: 'People', roles: ['ADMIN'] },
  { to: '/admin/catalog', label: 'Sessions & courses', roles: ['ADMIN', 'HALL_MANAGER'] },
  { to: '/admin/audit', label: 'Audit trail', roles: ['ADMIN'] },
]

function Layout({ children }) {
  const { user, signOut } = useAuth()
  const [menu, setMenu] = useState(false)
  const loc = useLocation()

  useEffect(() => { setMenu(false) }, [loc.pathname])

  return (
    <div className="shell">
      <header className="topbar">
        <div className="brand">
          Script Eval
          <small>Examination office</small>
        </div>
        <button
          className="nav-toggle"
          aria-label="Toggle navigation"
          aria-expanded={menu}
          onClick={() => setMenu((m) => !m)}
        >
          <span /><span /><span />
        </button>
        <nav className={menu ? 'open' : ''}>
          {links.filter((l) => user && l.roles.includes(user.role)).map((l) => (
            <NavLink key={l.to} to={l.to} className={({ isActive }) => (isActive ? 'active' : '')} end={l.to === '/scripts'}>
              {l.label}
            </NavLink>
          ))}
        </nav>
        <div className="who">
          <span className="who-name">{user?.full_name}</span>
          <span className="mono who-role">{user?.role.replace('_', ' ').toLowerCase()}</span>
          <NavLink to="/account" className="btn sm who-btn">Account</NavLink>
          <button className="btn sm who-btn" onClick={() => void signOut()}>Sign out</button>
        </div>
      </header>
      <main className="main">{children}</main>
    </div>
  )
}

export default function App() {
  const { user, loading } = useAuth()
  return (
    <Routes>
      <Route path="/login" element={user && !loading ? <Navigate to={home[user.role]} replace /> : <Login />} />
      <Route
        path="*"
        element={
          <Guard roles={['ADMIN', 'HALL_MANAGER', 'EVALUATOR', 'STUDENT']}>
            <Layout>
              <Routes>
                <Route path="/" element={<Navigate to={user ? home[user.role] : '/login'} replace />} />
                <Route path="/dashboard" element={<Guard roles={['ADMIN']}><Dashboard /></Guard>} />
                <Route path="/scripts" element={<Guard roles={['ADMIN', 'HALL_MANAGER']}><Scripts /></Guard>} />
                <Route path="/scripts/new" element={<Guard roles={['ADMIN', 'HALL_MANAGER']}><RegisterScript /></Guard>} />
                <Route path="/scripts/:id" element={<Guard roles={['ADMIN', 'HALL_MANAGER', 'EVALUATOR']}><ScriptDetail /></Guard>} />
                <Route path="/queue" element={<Guard roles={['EVALUATOR']}><Queue /></Guard>} />
                <Route path="/search" element={<Guard roles={['ADMIN', 'HALL_MANAGER', 'EVALUATOR']}><Search /></Guard>} />
                <Route path="/me" element={<Guard roles={['STUDENT']}><MyResults /></Guard>} />
                <Route path="/challenges" element={<Guard roles={['ADMIN', 'HALL_MANAGER']}><Challenges /></Guard>} />
                <Route path="/admin/users" element={<Guard roles={['ADMIN']}><Users /></Guard>} />
                <Route path="/admin/catalog" element={<Guard roles={['ADMIN', 'HALL_MANAGER']}><Catalog /></Guard>} />
                <Route path="/admin/audit" element={<Guard roles={['ADMIN']}><Audit /></Guard>} />
                <Route path="/account" element={<Account />} />
                <Route path="*" element={<Navigate to={user ? home[user.role] : '/login'} replace />} />
              </Routes>
            </Layout>
          </Guard>
        }
      />
    </Routes>
  )
}

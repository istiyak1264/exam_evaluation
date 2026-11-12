import { Link } from 'react-router-dom'
import { SkipLink } from './gov'

export function AuthShell({ title, children }) {
  return (
    <div className="auth-page">
      <SkipLink />
      <div className="auth-sheet">
        <header className="auth-head">
          <img className="crest" src="/pust-logo.png" alt="" />
          <div className="auth-id">
            <div className="org">University Examination Board</div>
            <div className="org-sub">Office of the Controller of Examinations · পরীক্ষা নিয়ন্ত্রণ বিভাগ</div>
          </div>
        </header>

        <div className="auth-rule" aria-hidden="true" />

        <main id="main" className="auth-main">
          <div className="auth-card">
            <img className="auth-mark" src="/pust-logo.png" alt="" aria-hidden="true" />
            <span className="auth-kicker">Official record access</span>
            <h1 className="auth-title">{title}</h1>
            {children}
          </div>
        </main>
      </div>
    </div>
  )
}

export function AuthLink({ to, children }) {
  return <Link to={to}>{children}</Link>
}

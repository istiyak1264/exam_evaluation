import { Link } from 'react-router-dom'
import { SkipLink } from './gov'

// Shared formal shell for the public auth screens: crest masthead on paper,
// no coloured background, a ruled card, and a footer note.
export function AuthShell({ title, intro, children, footer }) {
  return (
    <div className="auth-page">
      <SkipLink />
      <header className="auth-masthead">
        <div className="wrap">
          <img className="crest" src="/pust-logo.png" alt="" />
          <div>
            <div className="org">University Examination Board</div>
            <div className="org-sub">Office of the Controller of Examinations · পরীক্ষা নিয়ন্ত্রণ বিভাগ</div>
          </div>
        </div>
      </header>

      <main className="auth-body" id="main">
        <div className="auth-card">
          <span className="caption">{title}</span>
          {intro && <p className="hint" style={{ marginBottom: '1.4rem' }}>{intro}</p>}
          {children}
        </div>
        {footer && <div className="auth-foot">{footer}</div>}
      </main>

      <footer className="auth-strip">
        Anonymised double-blind marking · বেনামী দ্বি-নিরীক্ষক মূল্যায়ন
      </footer>
    </div>
  )
}

export function AuthLink({ to, children }) {
  return <Link to={to}>{children}</Link>
}

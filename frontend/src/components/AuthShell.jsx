import { Link } from 'react-router-dom'
import { SkipLink } from './gov'

// Official-document composition: the crest and board identity head a ruled
// sheet, the form sits on paper with a double hairline border, and the crest
// shows through as a watermark. One centred column at every width.
export function AuthShell({ title, intro, children, footer }) {
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
            {intro && <p className="auth-intro">{intro}</p>}
            {children}
          </div>
          {footer && <div className="auth-foot">{footer}</div>}
        </main>

        <footer className="auth-strip">
          <span>Anonymised double-blind marking · বেনামী দ্বি-নিরীক্ষক মূল্যায়ন</span>
          <span>Records retained for five years</span>
        </footer>
      </div>
    </div>
  )
}

export function AuthLink({ to, children }) {
  return <Link to={to}>{children}</Link>
}

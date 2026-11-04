import { Link } from 'react-router-dom'
import { SkipLink } from './gov'

// Formal auth shell: an information panel beside the form on wide screens,
// stacked on tablets and phones. No coloured page background.
export function AuthShell({ title, intro, children, footer }) {
  return (
    <div className="auth-page">
      <SkipLink />
      <div className="auth-split">
        <aside className="auth-aside">
          <img className="crest" src="/pust-logo.png" alt="" />
          <div className="org">University Examination Board</div>
          <div className="org-sub">Office of the Controller of Examinations · পরীক্ষা নিয়ন্ত্রণ বিভাগ</div>

          <ul className="auth-points">
            <li>Every script is marked twice, independently and without names.</li>
            <li>A third external examiner is appointed wherever the two marks differ by 13 or more.</li>
            <li>Students see their marks once the board upholds a challenge.</li>
          </ul>

          <p className="auth-note">Anonymised double-blind marking · বেনামী দ্বি-নিরীক্ষক মূল্যায়ন</p>
        </aside>

        <main className="auth-main" id="main">
          <div className="auth-card">
            <span className="caption">{title}</span>
            {intro && <p className="hint auth-intro">{intro}</p>}
            {children}
          </div>
          {footer && <div className="auth-foot">{footer}</div>}
        </main>
      </div>

      <footer className="auth-strip">
        Examination records are retained for five years · © {new Date().getFullYear()} University Examination Board
      </footer>
    </div>
  )
}

export function AuthLink({ to, children }) {
  return <Link to={to}>{children}</Link>
}

export function SkipLink() {
  return <a className="skip-link" href="#main">Skip to main content</a>
}

export function Breadcrumbs({ items = [] }) {
  if (!items.length) return null
  return (
    <nav className="breadcrumbs" aria-label="Breadcrumb">
      <ol>
        {items.map((it) => (
          <li key={it.label}>
            {it.href ? <a href={it.href}>{it.label}</a> : <span aria-current="page">{it.label}</span>}
          </li>
        ))}
      </ol>
    </nav>
  )
}

export function Caption({ children }) {
  return <span className="caption">{children}</span>
}

// rows: [{ key, value, action }]
export function SummaryList({ rows = [], className = '' }) {
  return (
    <dl className={`summary-list ${className}`}>
      {rows.map((r) => (
        <div className="summary-row" key={r.key}>
          <dt className="summary-key">{r.key}</dt>
          <dd className="summary-value">{r.value}</dd>
          {r.action && <dd className="summary-action">{r.action}</dd>}
        </div>
      ))}
    </dl>
  )
}

// items: [{ label, href, hint, status: 'done' | 'current' | 'todo', meta }]
export function TaskList({ items = [] }) {
  return (
    <ol className="task-list">
      {items.map((t) => (
        <li className={`task-item task-${t.status}`} key={t.label}>
          <span className="task-name">
            {t.href ? <a href={t.href}>{t.label}</a> : t.label}
            {t.hint && <span className="task-hint">{t.hint}</span>}
          </span>
          <span className="task-status">
            {t.status === 'done' ? 'Completed' : t.status === 'current' ? <strong>{t.meta ?? 'In progress'}</strong> : (t.meta ?? 'Not started')}
          </span>
        </li>
      ))}
    </ol>
  )
}

export function ErrorSummary({ errors = [], id = 'error-summary' }) {
  if (!errors.length) return null
  return (
    <div className="error-summary" id={id} tabIndex={-1} role="region" aria-labelledby={`${id}-title`}>
      <h2 className="error-summary-title" id={`${id}-title`}>There is a problem</h2>
      <ul className="error-list">
        {errors.map((e) => <li key={e}>{e}</li>)}
      </ul>
    </div>
  )
}

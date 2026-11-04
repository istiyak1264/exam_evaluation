import { useCallback, useEffect, useState, type ReactNode } from 'react'

export function Badge({ status }: { status: string }) {
  const map: Record<string, string> = {
    RESOLVED: 'ok', REGISTERED: '', ASSIGNED: 'info', EVAL_PENDING: 'warn',
    THIRD_REVIEW: 'warn', WITHDRAWN: 'danger', OPEN: 'warn', UPHELD: 'ok', DISMISSED: 'danger',
  }
  const label = status.replace(/_/g, ' ').toLowerCase()
  return <span className={`badge ${map[status] ?? ''}`}>{label}</span>
}

export function Grade({ grade }: { grade?: string | null }) {
  if (!grade) return <span className="mono">—</span>
  const cls = grade === 'F' ? 'danger' : grade === 'A' || grade === 'B' ? 'ok' : 'info'
  return <span className={`badge ${cls}`}>{grade}</span>
}

export function Note({ children, kind = 'warn' }: { children: ReactNode; kind?: 'warn' | 'info' | 'danger' }) {
  return <div className={`note ${kind === 'warn' ? '' : kind}`}>{children}</div>
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="card empty">{children}</div>
}

// useFetch runs a loader and exposes state; deps changes re-run.
export function useFetch<T>(loader: () => Promise<T>, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [nonce, setNonce] = useState(0)

  const reload = useCallback(() => setNonce((n) => n + 1), [])

  useEffect(() => {
    let alive = true
    setLoading(true)
    setError(null)
    loader()
      .then((d) => { if (alive) setData(d) })
      .catch((e) => { if (alive) setError(e?.message ?? 'request failed') })
      .finally(() => { if (alive) setLoading(false) })
    return () => { alive = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, nonce])

  return { data, error, loading, reload, setData }
}

export function Loading() {
  return <div className="card empty">Loading…</div>
}

export function ErrorBox({ message }: { message: string }) {
  return <Note kind="danger">{message}</Note>
}

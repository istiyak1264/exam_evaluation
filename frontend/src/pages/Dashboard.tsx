import { api } from '../api'
import { Loading, Note, useFetch } from '../components/ui'

export default function Dashboard() {
  const { data, error, loading } = useFetch(() => api.stats(), [])
  if (loading) return <Loading />
  if (error) return <Note kind="danger">{error}</Note>
  if (!data) return null

  const statuses = ['REGISTERED', 'ASSIGNED', 'EVAL_PENDING', 'THIRD_REVIEW', 'RESOLVED']
  const total = Object.values(data.scripts_by_status).reduce((a, b) => a + b, 0)

  return (
    <div className="stack">
      <div className="page-head">
        <div>
          <h1>Examination overview</h1>
          <p>Live position of every script in the double-blind pipeline. The third-examiner rule fires automatically at a gap of 13 marks or more.</p>
        </div>
      </div>

      <div className="grid">
        <div className="card kpi"><span className="n">{total}</span><span className="l">Scripts registered</span></div>
        <div className="card kpi"><span className="n">{data.results_resolved}</span><span className="l">Results resolved</span></div>
        <div className="card kpi"><span className="n">{data.third_review_rate.toFixed(1)}%</span><span className="l">Sent to third review</span></div>
        <div className="card kpi"><span className="n">{data.average_final_marks.toFixed(1)}</span><span className="l">Mean final mark</span></div>
        <div className="card kpi"><span className="n">{data.open_challenges}</span><span className="l">Open challenges</span></div>
      </div>

      <div className="card stack">
        <h2>Pipeline</h2>
        <table>
          <thead><tr><th>Stage</th><th>Scripts</th></tr></thead>
          <tbody>
            {statuses.map((s) => (
              <tr key={s}>
                <td>{s.replace(/_/g, ' ').toLowerCase()}</td>
                <td className="mono">{data.scripts_by_status[s] ?? 0}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="card stack">
        <h2>Staff accounts</h2>
        <table>
          <thead><tr><th>Role</th><th>Accounts</th></tr></thead>
          <tbody>
            {Object.entries(data.users_by_role).map(([role, n]) => (
              <tr key={role}><td>{role.replace('_', ' ').toLowerCase()}</td><td className="mono">{n}</td></tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

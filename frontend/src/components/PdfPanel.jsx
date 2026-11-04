import { useEffect, useState } from 'react'
import { api } from '../api'
import { ErrorBox, Loading } from './ui'

// PdfPanel mints a short-lived URL, then embeds the script in the browser viewer.
export function PdfPanel({ scriptId }) {
  const [url, setUrl] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    let alive = true
    setUrl(null)
    setError(null)
    api.pdfToken(scriptId)
      .then((r) => { if (alive) setUrl(r.url) })
      .catch((e) => { if (alive) setError(e?.message ?? 'could not open document') })
    return () => { alive = false }
  }, [scriptId])

  if (error) return <ErrorBox message={error} />
  if (!url) return <Loading />
  return (
    <div className="pdf-frame">
      <iframe src={url} title="Exam script" />
    </div>
  )
}

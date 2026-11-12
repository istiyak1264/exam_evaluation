import { useState } from 'react'

export default function PasswordField({ id, label, value, onChange, hint, hintId, autoComplete = 'current-password', minLength }) {
  const [show, setShow] = useState(false)
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      {hint && <span className="hint" id={hintId}>{hint}</span>}
      <div className="pw-wrap">
        <input
          id={id}
          type={show ? 'text' : 'password'}
          required
          minLength={minLength}
          autoComplete={autoComplete}
          aria-describedby={hint ? hintId : undefined}
          value={value}
          onChange={onChange}
        />
        <button type="button" className="pw-toggle" onClick={() => setShow((s) => !s)} aria-pressed={show}>
          {show ? 'Hide' : 'Show'}
        </button>
      </div>
    </div>
  )
}

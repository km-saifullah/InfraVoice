export function TextField({ label, error, hint, className = '', ...rest }) {
  return (
    <label className={`flex flex-col gap-1.5 ${className}`}>
      {label && <span className="text-sm font-medium text-mist-200">{label}</span>}
      <input
        className="rounded-md border border-ink-700 bg-ink-850 px-3 py-2 text-sm text-mist-100
          placeholder:text-mist-500 focus:border-signal"
        {...rest}
      />
      {hint && !error && <span className="text-xs text-mist-500">{hint}</span>}
      {error && <span className="text-xs text-danger">{error}</span>}
    </label>
  )
}

export function TextArea({ label, error, hint, className = '', ...rest }) {
  return (
    <label className={`flex flex-col gap-1.5 ${className}`}>
      {label && <span className="text-sm font-medium text-mist-200">{label}</span>}
      <textarea
        className="rounded-md border border-ink-700 bg-ink-850 px-3 py-2 text-sm text-mist-100
          placeholder:text-mist-500 focus:border-signal"
        {...rest}
      />
      {hint && !error && <span className="text-xs text-mist-500">{hint}</span>}
      {error && <span className="text-xs text-danger">{error}</span>}
    </label>
  )
}

const TONES = {
  error: 'border-danger/40 bg-danger-dim text-danger',
  warning: 'border-amber/40 bg-amber-dim text-amber',
  info: 'border-signal/30 bg-signal-dim text-signal',
}

export function Alert({ tone = 'info', children }) {
  return (
    <div className={`rounded-md border mt-4 px-4 py-3 text-sm ${TONES[tone]}`} role="status">
      {children}
    </div>
  )
}

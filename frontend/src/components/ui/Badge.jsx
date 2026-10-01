const STATUS_STYLES = {
  queued: 'bg-amber-dim text-amber',
  running: 'bg-amber-dim text-amber',
  succeeded: 'bg-success-dim text-success',
  failed: 'bg-danger-dim text-danger',
  received: 'bg-ink-800 text-mist-300',
  processing: 'bg-amber-dim text-amber',
  completed: 'bg-success-dim text-success',
  active: 'bg-success-dim text-success',
  archived: 'bg-ink-800 text-mist-400',
}

export function Badge({ status, children }) {
  const style = STATUS_STYLES[status] || 'bg-ink-800 text-mist-300'
  const isPulsing = status === 'running' || status === 'processing'

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${style}`}
    >
      {isPulsing && <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />}
      {children || status}
    </span>
  )
}

import { useState } from 'react'
import { Badge } from '../ui/Badge'
import { RunStepsLog } from './RunStepsLog'

function formatSummary(summary) {
  if (!summary) return null
  return `+${summary.add} ~${summary.change} -${summary.destroy}`
}

export function RunCard({ run }) {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <div className="rounded-lg border border-ink-800 bg-ink-900">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        className="flex w-full items-center justify-between px-4 py-3 text-left"
      >
        <div className="flex items-center gap-3">
          <span className="font-display text-sm font-semibold uppercase tracking-wide text-mist-200">
            {run.type}
          </span>
          <Badge status={run.status} />
          {run.summary && (
            <span className="font-mono text-xs text-mist-500">{formatSummary(run.summary)}</span>
          )}
        </div>
        <span className="text-xs text-mist-500">
          {new Date(run.created_at).toLocaleString()}
        </span>
      </button>

      {isOpen && (
        <div className="border-t border-ink-800 p-4">
          {run.error && (
            <p className="mb-3 rounded-md border border-danger/40 bg-danger-dim px-3 py-2 text-sm text-danger">
              {run.error}
            </p>
          )}
          <RunStepsLog steps={run.steps} />
        </div>
      )}
    </div>
  )
}

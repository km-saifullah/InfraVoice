import { Badge } from '../ui/Badge'

export function RunStepsLog({ steps }) {
  if (!steps || steps.length === 0) {
    return <p className="text-sm text-mist-500">No steps have started yet.</p>
  }

  return (
    <div className="flex flex-col gap-3">
      {steps.map((step, index) => (
        <div key={index} className="rounded-md border border-ink-800">
          <div className="flex items-center justify-between gap-3 border-b border-ink-800 px-3 py-2">
            <span className="min-w-0 break-all font-mono text-xs text-mist-300">{step.command}</span>
            <Badge status={step.status} />
          </div>
          {step.output && (
            <pre className="scrollbar-thin max-h-64 overflow-auto whitespace-pre-wrap px-3 py-2 font-mono text-xs text-mist-400">
              {step.output}
              {step.truncated && (
                <span className="mt-2 block text-amber">[output truncated]</span>
              )}
            </pre>
          )}
        </div>
      ))}
    </div>
  )
}

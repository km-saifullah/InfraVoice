export function EmptyState({ title, description, action }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-ink-700 px-6 py-12 text-center">
      <h3 className="font-display text-base font-semibold text-mist-200">{title}</h3>
      {description && <p className="max-w-sm text-sm text-mist-400">{description}</p>}
      {action}
    </div>
  )
}

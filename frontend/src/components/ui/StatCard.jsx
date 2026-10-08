// A single headline number with an optional one-line breakdown
// underneath it. Used on the Overview page.
export function StatCard({ label, value, detail }) {
  return (
    <div className="min-w-0 rounded-lg border border-ink-800 bg-ink-900 p-4 sm:p-5">
      <p className="truncate text-xs font-medium text-mist-400 sm:text-sm">{label}</p>
      <p className="mt-2 font-display text-3xl font-bold text-mist-100 sm:text-4xl">{value}</p>
      {detail && <p className="mt-2 text-xs text-mist-500 sm:text-sm">{detail}</p>}
    </div>
  )
}

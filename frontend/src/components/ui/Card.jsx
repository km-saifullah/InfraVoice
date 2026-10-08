export function Card({ title, subtitle, actions, className = '', children }) {
  return (
    <div className={`rounded-lg border border-ink-800 bg-ink-900 ${className}`}>
      {(title || actions) && (
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-ink-800 px-5 py-4">
          <div>
            {title && <h3 className="font-display text-base font-semibold text-mist-100">{title}</h3>}
            {subtitle && <p className="mt-0.5 text-sm text-mist-400">{subtitle}</p>}
          </div>
          {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
        </div>
      )}
      <div className="p-5">{children}</div>
    </div>
  )
}

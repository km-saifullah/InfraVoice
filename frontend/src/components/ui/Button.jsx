const VARIANTS = {
  primary: 'bg-signal text-ink-950 hover:bg-signal-600 disabled:bg-signal/40',
  secondary: 'bg-ink-800 text-mist-100 hover:bg-ink-700 disabled:text-mist-500',
  danger: 'bg-danger text-ink-950 hover:bg-danger/85 disabled:bg-danger/40',
  ghost: 'bg-transparent text-mist-200 hover:bg-ink-800 disabled:text-mist-500',
}

export function Button({
  variant = 'primary',
  isLoading = false,
  disabled,
  className = '',
  children,
  ...rest
}) {
  return (
    <button
      disabled={disabled || isLoading}
      className={`inline-flex items-center justify-center gap-2 rounded-md px-4 py-2 text-sm font-medium
        transition-colors disabled:cursor-not-allowed ${VARIANTS[variant]} ${className}`}
      {...rest}
    >
      {isLoading && (
        <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
      )}
      {children}
    </button>
  )
}

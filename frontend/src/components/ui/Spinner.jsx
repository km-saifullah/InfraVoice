export function Spinner({ className = 'h-5 w-5' }) {
  return (
    <span
      className={`inline-block animate-spin rounded-full border-2 border-mist-500 border-t-signal ${className}`}
    />
  )
}

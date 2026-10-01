export function Waveform({ levels, active = false, className = '' }) {
  return (
    <div className={`flex h-16 items-end justify-center gap-1 ${className}`}>
      {levels.map((level, index) => (
        <span
          key={index}
          className={`w-1.5 rounded-full transition-[height] duration-75 ${
            active ? 'bg-signal' : 'bg-ink-700'
          }`}
          style={{ height: `${Math.max(8, level * 100)}%` }}
        />
      ))}
    </div>
  )
}

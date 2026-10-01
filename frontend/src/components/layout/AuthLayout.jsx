const BAR_DELAYS = [0, 0.15, 0.3, 0.1, 0.25, 0.05, 0.2]

export function AuthLayout({ title, subtitle, children, footer }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-ink-950 px-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-5 flex h-12 items-end gap-1.5">
            {BAR_DELAYS.map((delay, index) => (
              <span
                key={index}
                className="w-2 rounded-full bg-signal"
                style={{
                  height: '100%',
                  animation: `infravoice-wave 1.1s ease-in-out ${delay}s infinite`,
                }}
              />
            ))}
          </div>
          <h1 className="font-display text-2xl font-bold text-mist-100">{title}</h1>
          <p className="mt-1 text-sm text-mist-400">{subtitle}</p>
        </div>

        <div className="rounded-lg border border-ink-800 bg-ink-900 p-6">{children}</div>

        {footer && <p className="mt-4 text-center text-sm text-mist-400">{footer}</p>}
      </div>

      <style>{`
        @keyframes infravoice-wave {
          0%, 100% { transform: scaleY(0.25); }
          50% { transform: scaleY(1); }
        }
      `}</style>
    </div>
  )
}

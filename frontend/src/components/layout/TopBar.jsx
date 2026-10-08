import { useAuth } from '../../context/AuthContext'
import { useMobileNav } from '../../context/MobileNavContext'
import { Button } from '../ui/Button'

export function TopBar({ title, subtitle }) {
  const { user, logout } = useAuth()
  const { toggle } = useMobileNav()

  return (
    <header className="flex items-center justify-between gap-3 border-b border-ink-800 bg-ink-950 px-4 py-4 sm:px-6">
      <div className="flex min-w-0 items-center gap-3">
        <button
          type="button"
          onClick={toggle}
          aria-label="Open menu"
          className="shrink-0 rounded-md p-1.5 text-mist-300 hover:bg-ink-800 hover:text-mist-100 md:hidden"
        >
          <svg viewBox="0 0 20 20" fill="none" className="h-5 w-5" stroke="currentColor" strokeWidth="1.75">
            <path d="M3 5h14M3 10h14M3 15h14" strokeLinecap="round" />
          </svg>
        </button>

        <div className="min-w-0">
          <h1 className="truncate font-display text-base font-semibold text-mist-100 sm:text-lg">
            {title}
          </h1>
          {subtitle && <p className="truncate text-sm text-mist-400">{subtitle}</p>}
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-2 sm:gap-4">
        <span className="hidden max-w-[12rem] truncate text-sm text-mist-400 sm:inline">
          {user?.email}
        </span>
        <Button variant="ghost" onClick={logout}>
          Log out
        </Button>
      </div>
    </header>
  )
}

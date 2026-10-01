import { useAuth } from '../../context/AuthContext'
import { Button } from '../ui/Button'

export function TopBar({ title, subtitle }) {
  const { user, logout } = useAuth()

  return (
    <header className="flex items-center justify-between border-b border-ink-800 bg-ink-950 px-6 py-4">
      <div>
        <h1 className="font-display text-lg font-semibold text-mist-100">{title}</h1>
        {subtitle && <p className="text-sm text-mist-400">{subtitle}</p>}
      </div>
      <div className="flex items-center gap-4">
        <span className="text-sm text-mist-400">{user?.email}</span>
        <Button variant="ghost" onClick={logout}>
          Log out
        </Button>
      </div>
    </header>
  )
}

import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { MobileNavProvider, useMobileNav } from '../../context/MobileNavContext'

function MobileBackdrop() {
  const { isOpen, close } = useMobileNav()

  if (!isOpen) return null

  return (
    <button
      type="button"
      aria-label="Close menu"
      onClick={close}
      className="fixed inset-0 z-30 bg-ink-950/70 backdrop-blur-sm md:hidden"
    />
  )
}

export function AppLayout() {
  return (
    <MobileNavProvider>
      <div className="flex h-screen bg-ink-950">
        <MobileBackdrop />
        <Sidebar />
        <div className="min-w-0 flex-1 overflow-y-auto">
          <Outlet />
        </div>
      </div>
    </MobileNavProvider>
  )
}

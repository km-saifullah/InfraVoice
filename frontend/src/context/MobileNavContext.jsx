import { createContext, useContext, useEffect, useState } from 'react'

// Sidebar and TopBar are siblings deep in the tree (TopBar is
// rendered per-page, inside <Outlet/>, not inside AppLayout itself),
// so a tiny context -- rather than prop drilling through every page
// -- is what lets TopBar's mobile menu button open the Sidebar drawer
// that AppLayout renders.
const MobileNavContext = createContext(null)

export function MobileNavProvider({ children }) {
  const [isOpen, setIsOpen] = useState(false)

  // Lock background scroll while the mobile drawer is open, and
  // always leave the lock cleared on unmount.
  useEffect(() => {
    if (!isOpen) return undefined

    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    return () => {
      document.body.style.overflow = previousOverflow
    }
  }, [isOpen])

  const value = {
    isOpen,
    open: () => setIsOpen(true),
    close: () => setIsOpen(false),
    toggle: () => setIsOpen((open) => !open),
  }

  return <MobileNavContext.Provider value={value}>{children}</MobileNavContext.Provider>
}

export function useMobileNav() {
  const context = useContext(MobileNavContext)

  if (!context) {
    throw new Error('useMobileNav must be used within a MobileNavProvider')
  }

  return context
}

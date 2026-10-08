import { useEffect, useState } from 'react'
import { Link, NavLink, useParams } from 'react-router-dom'
import { ProjectsApi } from '../../lib/endpoints'
import { useMobileNav } from '../../context/MobileNavContext'

export function Sidebar() {
  const { projectId } = useParams()
  const [projects, setProjects] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const { isOpen, close } = useMobileNav()

  useEffect(() => {
    let cancelled = false

    ProjectsApi.list()
      .then((data) => {
        if (!cancelled) setProjects(data.projects || [])
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [projectId])

  return (
    <aside
      className={`fixed inset-y-0 left-0 z-40 flex h-screen w-72 max-w-[85vw] shrink-0 flex-col
        border-r border-ink-800 bg-ink-900 transition-transform duration-200 ease-in-out
        md:static md:z-auto md:w-60 md:translate-x-0
        ${isOpen ? 'translate-x-0' : '-translate-x-full'}`}
    >
      <div className="flex items-center justify-between border-b border-ink-800 px-5 py-4">
        <Link
          to="/overview"
          onClick={close}
          className="font-display text-lg font-bold text-mist-100"
        >
          InfraVoice
        </Link>
        <button
          type="button"
          onClick={close}
          aria-label="Close menu"
          className="rounded-md p-1 text-mist-400 hover:bg-ink-800 hover:text-mist-100 md:hidden"
        >
          <svg viewBox="0 0 20 20" fill="none" className="h-5 w-5" stroke="currentColor" strokeWidth="1.75">
            <path d="M5 5l10 10M15 5L5 15" strokeLinecap="round" />
          </svg>
        </button>
      </div>

      <nav className="scrollbar-thin flex-1 overflow-y-auto px-3 py-4">
        <NavLink
          to="/overview"
          onClick={close}
          className={({ isActive }) =>
            `block rounded-md px-2 py-1.5 text-sm font-medium transition-colors ${
              isActive
                ? 'bg-signal-dim text-signal'
                : 'text-mist-300 hover:bg-ink-800 hover:text-mist-100'
            }`
          }
        >
          Overview
        </NavLink>

        <span className="mt-4 block px-2 text-xs font-medium text-mist-500">Projects</span>
        <div className="mt-2 flex flex-col gap-0.5">
          {isLoading && <span className="px-2 text-sm text-mist-500">Loading…</span>}
          {!isLoading && projects.length === 0 && (
            <span className="px-2 text-sm text-mist-500">No projects yet</span>
          )}
          {projects.map((project) => (
            <Link
              key={project.id}
              to={`/projects/${project.id}/voice`}
              onClick={close}
              className={`truncate rounded-md px-2 py-1.5 text-sm transition-colors ${
                project.id === projectId
                  ? 'bg-signal-dim text-signal'
                  : 'text-mist-300 hover:bg-ink-800 hover:text-mist-100'
              }`}
            >
              {project.name}
            </Link>
          ))}
        </div>
      </nav>

      <div className="border-t border-ink-800 p-3">
        <Link
          to="/projects"
          onClick={close}
          className="block rounded-md px-3 py-2 text-center text-sm font-medium text-mist-300 hover:bg-ink-800"
        >
          + New project
        </Link>
      </div>
    </aside>
  )
}

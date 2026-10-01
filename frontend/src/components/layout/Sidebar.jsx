import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ProjectsApi } from '../../lib/endpoints'

export function Sidebar() {
  const { projectId } = useParams()
  const [projects, setProjects] = useState([])
  const [isLoading, setIsLoading] = useState(true)

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
    <aside className="flex h-screen w-60 shrink-0 flex-col border-r border-ink-800 bg-ink-900">
      <div className="border-b border-ink-800 px-5 py-4">
        <Link to="/projects" className="font-display text-lg font-bold text-mist-100">
          InfraVoice
        </Link>
      </div>

      <nav className="scrollbar-thin flex-1 overflow-y-auto px-3 py-4">
        <span className="px-2 text-xs font-medium text-mist-500">Projects</span>
        <div className="mt-2 flex flex-col gap-0.5">
          {isLoading && <span className="px-2 text-sm text-mist-500">Loading…</span>}
          {!isLoading && projects.length === 0 && (
            <span className="px-2 text-sm text-mist-500">No projects yet</span>
          )}
          {projects.map((project) => (
            <Link
              key={project.id}
              to={`/projects/${project.id}/voice`}
              className={`rounded-md px-2 py-1.5 text-sm transition-colors ${
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
          className="block rounded-md px-3 py-2 text-center text-sm font-medium text-mist-300 hover:bg-ink-800"
        >
          + New project
        </Link>
      </div>
    </aside>
  )
}

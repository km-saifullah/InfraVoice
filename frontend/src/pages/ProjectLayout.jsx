import { useEffect, useState } from 'react'
import { NavLink, Outlet, useParams } from 'react-router-dom'
import { ProjectsApi } from '../lib/endpoints'
import { TopBar } from '../components/layout/TopBar'
import { Spinner } from '../components/ui/Spinner'
import { Alert } from '../components/ui/Alert'

const TABS = [
  { to: 'voice', label: 'Voice command' },
  { to: 'specifications', label: 'Specifications' },
  { to: 'terraform', label: 'Terraform runs' },
]

export function ProjectLayout() {
  const { projectId } = useParams()
  const [project, setProject] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    setProject(null)
    setError(null)
    ProjectsApi.get(projectId)
      .then((data) => setProject(data.project))
      .catch((caughtError) => setError(caughtError.message))
  }, [projectId])

  if (error) {
    return (
      <div className="p-6">
        <Alert tone="error">{error}</Alert>
      </div>
    )
  }

  if (!project) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Spinner className="h-6 w-6" />
      </div>
    )
  }

  return (
    <div>
      <TopBar title={project.name} subtitle={project.description} />

      <div className="border-b border-ink-800 px-6">
        <nav className="flex gap-6">
          {TABS.map((tab) => (
            <NavLink
              key={tab.to}
              to={tab.to}
              className={({ isActive }) =>
                `border-b-2 py-3 text-sm font-medium transition-colors ${
                  isActive
                    ? 'border-signal text-mist-100'
                    : 'border-transparent text-mist-400 hover:text-mist-200'
                }`
              }
            >
              {tab.label}
            </NavLink>
          ))}
        </nav>
      </div>

      <div className="px-6 py-6">
        <Outlet context={{ project }} />
      </div>
    </div>
  )
}

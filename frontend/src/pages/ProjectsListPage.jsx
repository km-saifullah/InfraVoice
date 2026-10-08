import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ProjectsApi } from '../lib/endpoints'
import { TopBar } from '../components/layout/TopBar'
import { Card } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { TextField, TextArea } from '../components/ui/TextField'
import { Alert } from '../components/ui/Alert'
import { EmptyState } from '../components/ui/EmptyState'
import { Spinner } from '../components/ui/Spinner'

export function ProjectsListPage() {
  const navigate = useNavigate()
  const [projects, setProjects] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const [loadError, setLoadError] = useState(null)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [createError, setCreateError] = useState(null)
  const [isCreating, setIsCreating] = useState(false)

  function loadProjects() {
    setIsLoading(true)
    ProjectsApi.list()
      .then((data) => setProjects(data.projects || []))
      .catch((error) => setLoadError(error.message))
      .finally(() => setIsLoading(false))
  }

  useEffect(loadProjects, [])

  async function handleCreate(event) {
    event.preventDefault()
    setCreateError(null)
    setIsCreating(true)

    try {
      const { project } = await ProjectsApi.create(name, description)
      setName('')
      setDescription('')
      navigate(`/projects/${project.id}/voice`)
    } catch (error) {
      setCreateError(error.message)
    } finally {
      setIsCreating(false)
    }
  }

  return (
    <div>
      <TopBar title="Projects" subtitle="A project groups one environment's commands, specs and infrastructure." />

      <div className="mx-auto grid max-w-5xl gap-6 px-4 py-6 sm:px-6 sm:py-8 lg:grid-cols-[1fr_320px]">
        <div>
          {isLoading && (
            <div className="flex justify-center py-16">
              <Spinner className="h-6 w-6" />
            </div>
          )}

          {!isLoading && loadError && <Alert tone="error">{loadError}</Alert>}

          {!isLoading && !loadError && projects.length === 0 && (
            <EmptyState
              title="No projects yet"
              description="Create your first project to start issuing voice commands."
            />
          )}

          {!isLoading && projects.length > 0 && (
            <div className="grid gap-3 sm:grid-cols-2">
              {projects.map((project) => (
                <button
                  key={project.id}
                  onClick={() => navigate(`/projects/${project.id}/voice`)}
                  className="rounded-lg border border-ink-800 bg-ink-900 p-4 text-left transition-colors hover:border-signal/50"
                >
                  <h3 className="font-display text-base font-semibold text-mist-100">{project.name}</h3>
                  {project.description && (
                    <p className="mt-1 text-sm text-mist-400">{project.description}</p>
                  )}
                  <span className="mt-3 inline-block text-xs text-mist-500">{project.status}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        <Card title="New project">
          <form onSubmit={handleCreate} className="flex flex-col gap-4">
            <TextField
              label="Name"
              required
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="staging-infra"
            />
            <TextArea
              label="Description"
              rows={3}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="Optional"
            />
            {createError && <Alert tone="error">{createError}</Alert>}
            <Button type="submit" isLoading={isCreating}>
              Create project
            </Button>
          </form>
        </Card>
      </div>
    </div>
  )
}

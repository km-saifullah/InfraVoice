import { useEffect, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { InfrastructureApi, TerraformGenerateApi } from '../lib/endpoints'
import { SpecSummary } from '../components/specification/SpecSummary'
import { Card } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'
import { Alert } from '../components/ui/Alert'
import { EmptyState } from '../components/ui/EmptyState'
import { Spinner } from '../components/ui/Spinner'

export function SpecificationsPage() {
  const { project } = useOutletContext()
  const navigate = useNavigate()

  const [specs, setSpecs] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const [selectedId, setSelectedId] = useState(null)

  const [validation, setValidation] = useState(null)
  const [region, setRegion] = useState('us-east-1')
  const [generatedFiles, setGeneratedFiles] = useState(null)
  const [actionError, setActionError] = useState(null)
  const [isBusy, setIsBusy] = useState(false)

  function loadSpecs() {
    setIsLoading(true)
    InfrastructureApi.list(project.id)
      .then((data) => setSpecs(data.specifications || []))
      .finally(() => setIsLoading(false))
  }

  useEffect(loadSpecs, [project.id])

  const selected = specs.find((spec) => spec.id === selectedId) || null

  function selectSpec(id) {
    setSelectedId(id)
    setValidation(null)
    setGeneratedFiles(null)
    setActionError(null)
  }

  async function handleValidate() {
    setIsBusy(true)
    setActionError(null)
    try {
      await InfrastructureApi.validate(project.id, selected.id)
      setValidation({ ok: true, message: 'This specification is valid.' })
    } catch (error) {
      setValidation({ ok: false, message: error.message })
    } finally {
      setIsBusy(false)
    }
  }

  async function handleGenerate() {
    setIsBusy(true)
    setActionError(null)
    try {
      const { files } = await TerraformGenerateApi.generate(project.id, selected.id, region)
      setGeneratedFiles(files)
    } catch (error) {
      setActionError(error.message)
    } finally {
      setIsBusy(false)
    }
  }

  async function handleDelete() {
    if (!window.confirm('Delete this specification? This cannot be undone.')) return
    setIsBusy(true)
    try {
      await InfrastructureApi.remove(project.id, selected.id)
      setSelectedId(null)
      loadSpecs()
    } catch (error) {
      setActionError(error.message)
    } finally {
      setIsBusy(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-48 items-center justify-center">
        <Spinner className="h-6 w-6" />
      </div>
    )
  }

  if (specs.length === 0) {
    return (
      <EmptyState
        title="No specifications saved yet"
        description="Use the Voice command tab to turn a spoken or typed command into a specification, then save it here."
      />
    )
  }

  return (
    <div className="grid gap-6 lg:grid-cols-[280px_1fr]">
      <div className="flex flex-col gap-2">
        {specs.map((spec) => (
          <button
            key={spec.id}
            onClick={() => selectSpec(spec.id)}
            className={`rounded-md border px-3 py-2 text-left text-sm transition-colors ${
              spec.id === selectedId
                ? 'border-signal/60 bg-signal-dim text-signal'
                : 'border-ink-800 bg-ink-900 text-mist-300 hover:border-ink-700'
            }`}
          >
            <div className="font-medium">v{spec.version} · {spec.provider}</div>
            <div className="text-xs text-mist-500">
              {new Date(spec.created_at).toLocaleString()}
            </div>
          </button>
        ))}
      </div>

      {!selected && (
        <EmptyState title="Select a specification" description="Pick one from the list to review it." />
      )}

      {selected && (
        <Card
          title={`Specification v${selected.version}`}
          actions={
            <>
              <Button variant="secondary" onClick={handleValidate} isLoading={isBusy}>
                Validate
              </Button>
              <Button variant="danger" onClick={handleDelete} isLoading={isBusy}>
                Delete
              </Button>
            </>
          }
        >
          <SpecSummary spec={selected} />

          {validation && (
            <Alert tone={validation.ok ? 'info' : 'error'} className="mt-4">
              {validation.message}
            </Alert>
          )}

          <div className="mt-6 border-t border-ink-800 pt-5">
            <p className="mb-2 text-xs font-medium uppercase text-mist-500">
              Preview Terraform files
            </p>
            <div className="flex gap-2">
              <TextField
                className="w-48"
                value={region}
                onChange={(event) => setRegion(event.target.value)}
                placeholder="us-east-1"
              />
              <Button variant="secondary" onClick={handleGenerate} isLoading={isBusy}>
                Generate preview
              </Button>
              <Button
                onClick={() =>
                  navigate(`/projects/${project.id}/terraform`, { state: { specId: selected.id, region } })
                }
              >
                Go to Terraform runs
              </Button>
            </div>

            {actionError && (
              <Alert tone="error" className="mt-3">
                {actionError}
              </Alert>
            )}

            {generatedFiles && (
              <div className="mt-4 flex flex-col gap-3">
                {Object.entries(generatedFiles).map(([filename, content]) => (
                  <div key={filename} className="rounded-md border border-ink-800">
                    <div className="border-b border-ink-800 px-3 py-1.5 font-mono text-xs text-mist-400">
                      {filename}
                    </div>
                    <pre className="scrollbar-thin max-h-80 overflow-auto whitespace-pre-wrap px-3 py-2 font-mono text-xs text-mist-300">
                      {content}
                    </pre>
                  </div>
                ))}
              </div>
            )}
          </div>
        </Card>
      )}
    </div>
  )
}

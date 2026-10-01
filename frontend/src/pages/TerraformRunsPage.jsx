import { useEffect, useMemo, useState } from 'react'
import { useLocation, useOutletContext } from 'react-router-dom'
import { InfrastructureApi, TerraformRunsApi } from '../lib/endpoints'
import { useTerraformRun } from '../hooks/useTerraformRun'
import { Card } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'
import { Alert } from '../components/ui/Alert'
import { EmptyState } from '../components/ui/EmptyState'
import { Badge } from '../components/ui/Badge'
import { RunStepsLog } from '../components/terraform/RunStepsLog'
import { RunCard } from '../components/terraform/RunCard'

export function TerraformRunsPage() {
  const { project } = useOutletContext()
  const location = useLocation()

  const [specs, setSpecs] = useState([])
  const [selectedSpecId, setSelectedSpecId] = useState(location.state?.specId || '')
  const [region, setRegion] = useState(location.state?.region || 'us-east-1')

  const [runs, setRuns] = useState([])
  const [activeRunId, setActiveRunId] = useState(null)
  const [startError, setStartError] = useState(null)

  const [confirmApply, setConfirmApply] = useState(false)
  const [confirmDestroy, setConfirmDestroy] = useState(false)

  const { run: activeRun } = useTerraformRun(project.id, activeRunId)

  useEffect(() => {
    InfrastructureApi.list(project.id).then((data) => {
      const list = data.specifications || []
      setSpecs(list)
      if (!selectedSpecId && list.length > 0) setSelectedSpecId(list[0].id)
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project.id])

  function loadRuns() {
    TerraformRunsApi.list(project.id).then((data) => setRuns(data.runs || []))
  }

  useEffect(loadRuns, [project.id])

  useEffect(() => {
    setActiveRunId(null)
    setConfirmApply(false)
    setConfirmDestroy(false)
  }, [selectedSpecId])

  useEffect(() => {
    if (activeRun && (activeRun.status === 'succeeded' || activeRun.status === 'failed')) {
      loadRuns()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeRun?.status])

  const latestSucceededPlan = useMemo(() => {
    const candidates = [activeRun, ...runs].filter(Boolean)
    return candidates.find(
      (run) =>
        run.type === 'plan' &&
        run.status === 'succeeded' &&
        run.specification_id === selectedSpecId,
    )
  }, [activeRun, runs, selectedSpecId])

  async function startRun(body) {
    setStartError(null)
    try {
      const { run } = await TerraformRunsApi.start(project.id, body)
      setActiveRunId(run.id)
    } catch (error) {
      setStartError(error.message)
    }
  }

  const isBusy = activeRun && activeRun.status !== 'succeeded' && activeRun.status !== 'failed'

  if (specs.length === 0) {
    return (
      <EmptyState
        title="No specifications to run yet"
        description="Save a specification from the Voice command tab first."
      />
    )
  }

  return (
    <div className="grid gap-6">
      <Card title="Run Terraform">
        <div className="flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1.5">
            <span className="text-sm font-medium text-mist-200">Specification</span>
            <select
              value={selectedSpecId}
              onChange={(event) => setSelectedSpecId(event.target.value)}
              className="rounded-md border border-ink-700 bg-ink-850 px-3 py-2 text-sm text-mist-100"
            >
              {specs.map((spec) => (
                <option key={spec.id} value={spec.id}>
                  v{spec.version} · {new Date(spec.created_at).toLocaleDateString()}
                </option>
              ))}
            </select>
          </label>

          <TextField
            label="AWS region"
            value={region}
            onChange={(event) => setRegion(event.target.value)}
            className="w-44"
          />

          <Button
            variant="secondary"
            disabled={isBusy}
            onClick={() => startRun({ specification_id: selectedSpecId, region, type: 'validate' })}
          >
            Validate
          </Button>
          <Button
            disabled={isBusy}
            onClick={() => startRun({ specification_id: selectedSpecId, region, type: 'plan' })}
          >
            Plan
          </Button>
        </div>

        {startError && (
          <Alert tone="error" className="mt-4">
            {startError}
          </Alert>
        )}

        {activeRun && (
          <div className="mt-5 border-t border-ink-800 pt-4">
            <div className="mb-3 flex items-center gap-3">
              <span className="font-display text-sm font-semibold uppercase text-mist-200">
                {activeRun.type}
              </span>
              <Badge status={activeRun.status} />
              {activeRun.summary && (
                <span className="font-mono text-xs text-mist-500">
                  +{activeRun.summary.add} ~{activeRun.summary.change} -{activeRun.summary.destroy}
                </span>
              )}
            </div>
            {activeRun.error && (
              <Alert tone="error" className="mb-3">
                {activeRun.error}
              </Alert>
            )}
            <RunStepsLog steps={activeRun.steps} />
          </div>
        )}
      </Card>

      <Card
        title="Apply"
        subtitle="Applies the exact plan you reviewed above. Creates real AWS resources."
      >
        {!latestSucceededPlan ? (
          <p className="text-sm text-mist-400">Run a plan for this specification first.</p>
        ) : (
          <div className="flex flex-col gap-3">
            <label className="flex items-start gap-2 text-sm text-mist-300">
              <input
                type="checkbox"
                checked={confirmApply}
                onChange={(event) => setConfirmApply(event.target.checked)}
                className="mt-0.5"
              />
              I understand this will create real AWS resources and may incur cost.
            </label>
            <Button
              variant="primary"
              disabled={!confirmApply || isBusy}
              onClick={() => {
                startRun({ type: 'apply', plan_run_id: latestSucceededPlan.id, confirm: true })
                setConfirmApply(false)
              }}
              className="w-fit"
            >
              Apply plan {latestSucceededPlan.id.slice(-6)}
            </Button>
          </div>
        )}
      </Card>

      <Card
        title="Destroy"
        subtitle="Tears down every resource this specification's workspace manages."
      >
        <div className="flex flex-col gap-3">
          <label className="flex items-start gap-2 text-sm text-mist-300">
            <input
              type="checkbox"
              checked={confirmDestroy}
              onChange={(event) => setConfirmDestroy(event.target.checked)}
              className="mt-0.5"
            />
            I understand this will permanently delete real AWS resources.
          </label>
          <Button
            variant="danger"
            disabled={!confirmDestroy || isBusy}
            onClick={() => {
              startRun({ specification_id: selectedSpecId, region, type: 'destroy', confirm: true })
              setConfirmDestroy(false)
            }}
            className="w-fit"
          >
            Destroy infrastructure
          </Button>
        </div>
      </Card>

      <div>
        <h3 className="mb-3 font-display text-sm font-semibold text-mist-300">Run history</h3>
        <div className="flex flex-col gap-2">
          {runs.length === 0 && <p className="text-sm text-mist-500">No runs yet.</p>}
          {runs.map((run) => (
            <RunCard key={run.id} run={run} />
          ))}
        </div>
      </div>
    </div>
  )
}

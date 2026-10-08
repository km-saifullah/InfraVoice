import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { OverviewApi } from '../lib/endpoints'
import { TopBar } from '../components/layout/TopBar'
import { Card } from '../components/ui/Card'
import { StatCard } from '../components/ui/StatCard'
import { Badge } from '../components/ui/Badge'
import { Button } from '../components/ui/Button'
import { Alert } from '../components/ui/Alert'
import { EmptyState } from '../components/ui/EmptyState'
import { Spinner } from '../components/ui/Spinner'

const RUN_STATUSES = ['queued', 'running', 'succeeded', 'failed']
const RUN_TYPES = ['validate', 'plan', 'apply', 'destroy']

// Turns a { key: count } map into rows. When `order` is given those
// keys always appear (as 0 if absent) so the layout is stable; any
// other keys the backend returns are appended, never dropped.
function toRows(counts, order) {
  const map = counts || {}

  if (!order) {
    return Object.entries(map).sort((a, b) => b[1] - a[1])
  }

  const extra = Object.keys(map).filter((key) => !order.includes(key))

  return [...order, ...extra].map((key) => [key, map[key] || 0])
}

function BreakdownRows({ title, counts, order, renderLabel }) {
  const rows = toRows(counts, order)
  const max = Math.max(1, ...rows.map(([, count]) => count))

  return (
    <div>
      <h4 className="mb-2 text-xs font-medium text-mist-500">{title}</h4>
      {rows.length === 0 ? (
        <p className="text-sm text-mist-500">No data yet.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {rows.map(([key, count]) => (
            <li key={key} className="flex flex-col gap-1">
              <div className="flex items-center justify-between gap-3 text-sm">
                <span className="min-w-0 truncate text-mist-300">
                  {renderLabel ? renderLabel(key) : key}
                </span>
                <span className="font-mono text-mist-100">{count}</span>
              </div>
              <div className="h-1 overflow-hidden rounded-full bg-ink-800">
                <div
                  className="h-full rounded-full bg-signal/70"
                  style={{ width: `${(count / max) * 100}%` }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export function OverviewPage() {
  const navigate = useNavigate()
  const [overview, setOverview] = useState(null)
  const [error, setError] = useState(null)
  const [isLoading, setIsLoading] = useState(true)

  const load = useCallback(() => {
    setIsLoading(true)
    setError(null)

    OverviewApi.get()
      .then((data) => setOverview(data.overview))
      .catch((caughtError) => setError(caughtError.message))
      .finally(() => setIsLoading(false))
  }, [])

  useEffect(load, [load])

  return (
    <div>
      <TopBar title="Overview" subtitle="A snapshot across every project you own." />

      <div className="mx-auto max-w-6xl px-4 py-6 sm:px-6 sm:py-8">
        {isLoading && !overview && (
          <div className="flex justify-center py-16">
            <Spinner className="h-6 w-6" />
          </div>
        )}

        {error && (
          <div className="flex flex-col gap-3">
            <Alert tone="error">{error}</Alert>
            <Button variant="secondary" onClick={load} className="w-fit">
              Try again
            </Button>
          </div>
        )}

        {!error && overview && overview.projects.total === 0 && (
          <EmptyState
            title="Nothing here yet"
            description="Create your first project, then speak a command to start building infrastructure."
            action={<Button onClick={() => navigate('/projects')}>Create a project</Button>}
          />
        )}

        {!error && overview && overview.projects.total > 0 && (
          <div className="flex flex-col gap-6">
            <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
              <StatCard
                label="Projects"
                value={overview.projects.total}
                detail={`${overview.projects.active} active, ${overview.projects.archived} archived`}
              />
              <StatCard
                label="Specifications"
                value={overview.infrastructure.specifications}
                detail="Saved infrastructure plans"
              />
              <StatCard
                label="Commands"
                value={overview.commands.total}
                detail={`${overview.commands.by_source?.voice || 0} voice, ${
                  overview.commands.by_source?.text || 0
                } text`}
              />
              <StatCard
                label="Terraform runs"
                value={overview.terraform.total}
                detail={`${overview.terraform.by_status?.succeeded || 0} succeeded, ${
                  overview.terraform.by_status?.failed || 0
                } failed`}
              />
            </div>

            <div className="grid gap-6 lg:grid-cols-2">
              <Card
                title="Resources in your specifications"
                subtitle="What your saved specs define. This is not a live view of your AWS account."
              >
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  {[
                    ['VPC', overview.infrastructure.resources.vpc],
                    ['EC2', overview.infrastructure.resources.ec2],
                    ['S3', overview.infrastructure.resources.s3],
                    ['SNS', overview.infrastructure.resources.sns],
                  ].map(([label, count]) => (
                    <div key={label} className="rounded-md border border-ink-800 px-3 py-3">
                      <p className="text-xs text-mist-500">{label}</p>
                      <p className="mt-1 font-display text-2xl font-semibold text-mist-100">
                        {count}
                      </p>
                    </div>
                  ))}
                </div>
              </Card>

              <Card title="Commands" subtitle="Everything you've spoken or typed.">
                <div className="grid gap-5 sm:grid-cols-2">
                  <BreakdownRows
                    title="By status"
                    counts={overview.commands.by_status}
                    renderLabel={(key) => <Badge status={key} />}
                  />
                  <BreakdownRows title="By source" counts={overview.commands.by_source} />
                </div>
              </Card>
            </div>

            <Card title="Terraform runs" subtitle="Across all projects.">
              <div className="grid gap-6 md:grid-cols-3">
                <BreakdownRows
                  title="By status"
                  counts={overview.terraform.by_status}
                  order={RUN_STATUSES}
                  renderLabel={(key) => <Badge status={key} />}
                />
                <BreakdownRows
                  title="By type"
                  counts={overview.terraform.by_type}
                  order={RUN_TYPES}
                />
                <BreakdownRows title="By AWS region" counts={overview.terraform.by_region} />
              </div>
            </Card>
          </div>
        )}
      </div>
    </div>
  )
}

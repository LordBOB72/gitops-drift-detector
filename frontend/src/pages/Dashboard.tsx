import { useDrift, useTriggerDetection } from '../hooks/useCluster'
import { DriftTable } from '../components/DriftTable'
import { DriftHistoryChart } from '../components/DriftHistoryChart'
import { SeverityBadge } from '../components/SeverityBadge'
import type { Severity } from '../types'

interface Props {
  clusterId: string | null
}

export function Dashboard({ clusterId }: Props) {
  const { data: events = [], isLoading, error } = useDrift(clusterId)
  const trigger = useTriggerDetection(clusterId ?? '')

  if (!clusterId) {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-3 text-gray-500">
        <span className="text-4xl">⎈</span>
        <p className="font-mono text-sm">No cluster selected. Add one under Clusters.</p>
      </div>
    )
  }

  if (isLoading) {
    return <div className="font-mono text-sm text-gray-500 animate-pulse">Loading drift state…</div>
  }

  if (error) {
    return <div className="font-mono text-sm text-red-400">Error: {String(error)}</div>
  }

  const counts = events.reduce(
    (acc, ev) => { acc[ev.severity]++; return acc },
    { critical: 0, warning: 0, info: 0 } as Record<Severity, number>
  )

  return (
    <div className="space-y-6">
      {/* summary cards */}
      <div className="grid grid-cols-3 gap-4">
        {(['critical', 'warning', 'info'] as Severity[]).map((s) => (
          <div key={s} className="bg-surface-1 border border-surface-3 rounded-lg p-4 flex items-center justify-between">
            <div>
              <p className="text-xs text-gray-500 mb-1 uppercase tracking-wider">{s}</p>
              <p className="text-2xl font-mono font-medium">{counts[s]}</p>
            </div>
            <SeverityBadge severity={s} />
          </div>
        ))}
      </div>

      {/* drift history chart */}
      <div className="bg-surface-1 border border-surface-3 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-medium text-gray-300">Drift history — last 14 days</h2>
          <button
            onClick={() => trigger.mutate()}
            disabled={trigger.isPending}
            className="text-xs px-3 py-1.5 bg-surface-3 border border-surface-3 hover:border-accent-blue text-gray-300 rounded transition-colors disabled:opacity-50 font-mono"
          >
            {trigger.isPending ? 'detecting…' : '⟳ detect now'}
          </button>
        </div>
        <DriftHistoryChart events={events} />
      </div>

      {/* drift table */}
      <div className="bg-surface-1 border border-surface-3 rounded-lg p-4">
        <h2 className="text-sm font-medium text-gray-300 mb-4">
          Active drift — {events.length} resource{events.length !== 1 ? 's' : ''}
        </h2>
        <DriftTable events={events} clusterId={clusterId} />
      </div>
    </div>
  )
}

import { useState } from 'react'
import clsx from 'clsx'
import type { DriftEvent } from '../types'
import { SeverityBadge } from './SeverityBadge'
import { useReconcile } from '../hooks/useCluster'
import { formatDistanceToNow } from 'date-fns'

interface Props {
  events: DriftEvent[]
  clusterId: string
}

export function DriftTable({ events, clusterId }: Props) {
  const [expanded, setExpanded] = useState<string | null>(null)
  const reconcile = useReconcile(clusterId)

  if (events.length === 0) {
    return (
      <div className="text-center py-16 text-surface-3 font-mono text-sm">
        ✓ No drift detected
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-surface-3 overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-surface-3 bg-surface-2 text-left">
            <th className="px-4 py-3 font-medium text-gray-400">Resource</th>
            <th className="px-4 py-3 font-medium text-gray-400">Namespace</th>
            <th className="px-4 py-3 font-medium text-gray-400">Type</th>
            <th className="px-4 py-3 font-medium text-gray-400">Severity</th>
            <th className="px-4 py-3 font-medium text-gray-400">Detected</th>
            <th className="px-4 py-3 font-medium text-gray-400">Actions</th>
          </tr>
        </thead>
        <tbody>
          {events.map((ev) => (
            <>
              <tr
                key={ev.id}
                className={clsx(
                  'border-b border-surface-3 cursor-pointer hover:bg-surface-2 transition-colors',
                  expanded === ev.id && 'bg-surface-2'
                )}
                onClick={() => setExpanded(expanded === ev.id ? null : ev.id)}
              >
                <td className="px-4 py-3 font-mono text-accent-cyan">
                  {ev.resource_kind}/{ev.resource_name}
                </td>
                <td className="px-4 py-3 text-gray-400 font-mono">{ev.namespace || '—'}</td>
                <td className="px-4 py-3">
                  <span className="font-mono text-xs text-gray-300">{ev.drift_type}</span>
                </td>
                <td className="px-4 py-3">
                  <SeverityBadge severity={ev.severity} />
                </td>
                <td className="px-4 py-3 text-gray-400 text-xs">
                  {formatDistanceToNow(new Date(ev.detected_at), { addSuffix: true })}
                </td>
                <td className="px-4 py-3">
                  {ev.drift_type !== 'unexpected' && ev.desired_state && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        reconcile.mutate({
                          resource_kind: ev.resource_kind,
                          resource_name: ev.resource_name,
                          namespace: ev.namespace,
                          manifest: JSON.stringify(ev.desired_state),
                        })
                      }}
                      disabled={reconcile.isPending}
                      className="text-xs px-3 py-1 bg-accent-blue/20 text-accent-blue border border-accent-blue/40 rounded hover:bg-accent-blue/30 transition-colors disabled:opacity-50"
                    >
                      Reconcile
                    </button>
                  )}
                </td>
              </tr>
              {expanded === ev.id && (
                <tr key={`${ev.id}-expanded`} className="bg-surface-1">
                  <td colSpan={6} className="px-4 py-4">
                    <DiffView event={ev} />
                  </td>
                </tr>
              )}
            </>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function DiffView({ event }: { event: DriftEvent }) {
  const diff = event.diff ? String(event.diff) : null
  if (!diff) {
    return (
      <pre className="text-xs font-mono text-gray-400 whitespace-pre-wrap">
        {JSON.stringify(event.live_state ?? event.desired_state, null, 2)}
      </pre>
    )
  }

  // diff is a unified diff string from go-cmp
  const lines = diff.split('\n')
  return (
    <pre className="text-xs font-mono leading-5 overflow-x-auto">
      {lines.map((line, i) => (
        <span
          key={i}
          className={clsx(
            'block',
            line.startsWith('+') && 'text-accent-green bg-green-500/10',
            line.startsWith('-') && 'text-red-400 bg-red-500/10',
            !line.startsWith('+') && !line.startsWith('-') && 'text-gray-500'
          )}
        >
          {line}
        </span>
      ))}
    </pre>
  )
}

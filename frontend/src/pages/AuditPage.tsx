import { useAudit } from '../hooks/useCluster'
import { formatDistanceToNow } from 'date-fns'

interface Props {
  clusterId: string | null
}

export function AuditPage({ clusterId }: Props) {
  const { data: entries = [], isLoading } = useAudit(clusterId)

  if (!clusterId) {
    return <p className="text-sm text-gray-500 font-mono">Select a cluster to view audit log.</p>
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-medium">Audit Log</h1>
      <div className="bg-surface-1 border border-surface-3 rounded-lg overflow-hidden">
        {isLoading && (
          <p className="p-4 text-sm text-gray-500 font-mono animate-pulse">Loading…</p>
        )}
        {!isLoading && entries.length === 0 && (
          <p className="p-4 text-sm text-gray-500 font-mono">No audit entries yet.</p>
        )}
        {entries.map((e, i) => (
          <div
            key={e.id}
            className={`px-4 py-3 flex items-start gap-4 text-sm border-b border-surface-3 last:border-0 ${
              i % 2 === 0 ? '' : 'bg-surface-2/30'
            }`}
          >
            <span className="font-mono text-xs text-gray-500 whitespace-nowrap pt-0.5">
              {formatDistanceToNow(new Date(e.created_at), { addSuffix: true })}
            </span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-mono text-xs px-1.5 py-0.5 bg-surface-3 rounded text-accent-cyan">
                  {e.action}
                </span>
                <span className="text-xs text-gray-400">by {e.actor}</span>
              </div>
              {e.resource && (
                <p className="text-xs text-gray-500 font-mono mt-0.5 truncate">{e.resource}</p>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

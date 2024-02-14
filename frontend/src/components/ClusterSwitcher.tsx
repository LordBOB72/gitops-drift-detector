import type { Cluster } from '../types'

interface Props {
  clusters: Cluster[]
  active: string | null
  onChange: (id: string) => void
}

export function ClusterSwitcher({ clusters, active, onChange }: Props) {
  if (clusters.length === 0) {
    return (
      <span className="text-xs text-gray-500 font-mono">no clusters registered</span>
    )
  }

  return (
    <select
      value={active ?? ''}
      onChange={(e) => onChange(e.target.value)}
      className="bg-surface-2 border border-surface-3 text-sm text-gray-300 rounded px-3 py-1.5 font-mono focus:outline-none focus:border-accent-blue"
    >
      {clusters.map((c) => (
        <option key={c.id} value={c.id}>
          {c.name}
        </option>
      ))}
    </select>
  )
}

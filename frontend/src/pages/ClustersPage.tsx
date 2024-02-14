import { useState } from 'react'
import { useClusters } from '../hooks/useCluster'
import { api } from '../lib/api'
import { useQueryClient } from '@tanstack/react-query'

export function ClustersPage() {
  const { data: clusters = [], isLoading } = useClusters()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [kubeconfig, setKubeconfig] = useState('')
  const [adding, setAdding] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  async function addCluster() {
    if (!name.trim()) return
    setAdding(true)
    setErr(null)
    try {
      await api.clusters.add(name.trim(), kubeconfig.trim())
      setName('')
      setKubeconfig('')
      qc.invalidateQueries({ queryKey: ['clusters'] })
    } catch (e) {
      setErr(String(e))
    } finally {
      setAdding(false)
    }
  }

  async function removeCluster(id: string) {
    await api.clusters.remove(id)
    qc.invalidateQueries({ queryKey: ['clusters'] })
  }

  return (
    <div className="max-w-2xl space-y-6">
      <h1 className="text-lg font-medium">Clusters</h1>

      {/* add cluster form */}
      <div className="bg-surface-1 border border-surface-3 rounded-lg p-5 space-y-4">
        <h2 className="text-sm font-medium text-gray-400">Register cluster</h2>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Name</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="production-us-east"
            className="w-full bg-surface-0 border border-surface-3 rounded px-3 py-2 text-sm font-mono focus:outline-none focus:border-accent-blue"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">
            Kubeconfig <span className="text-gray-600">(leave blank for in-cluster)</span>
          </label>
          <textarea
            value={kubeconfig}
            onChange={(e) => setKubeconfig(e.target.value)}
            rows={6}
            placeholder="apiVersion: v1&#10;kind: Config&#10;..."
            className="w-full bg-surface-0 border border-surface-3 rounded px-3 py-2 text-xs font-mono focus:outline-none focus:border-accent-blue resize-none"
          />
        </div>
        {err && <p className="text-xs text-red-400 font-mono">{err}</p>}
        <button
          onClick={addCluster}
          disabled={adding || !name.trim()}
          className="px-4 py-2 bg-accent-blue/20 text-accent-blue border border-accent-blue/40 rounded text-sm hover:bg-accent-blue/30 transition-colors disabled:opacity-50"
        >
          {adding ? 'Registering…' : 'Register'}
        </button>
      </div>

      {/* cluster list */}
      <div className="space-y-2">
        {isLoading && <p className="text-sm text-gray-500 font-mono animate-pulse">Loading…</p>}
        {clusters.map((c) => (
          <div
            key={c.id}
            className="flex items-center justify-between bg-surface-1 border border-surface-3 rounded-lg px-4 py-3"
          >
            <div>
              <p className="font-mono text-sm text-accent-cyan">{c.name}</p>
              <p className="text-xs text-gray-500 font-mono">{c.id}</p>
            </div>
            <button
              onClick={() => removeCluster(c.id)}
              className="text-xs text-gray-500 hover:text-red-400 transition-colors font-mono"
            >
              remove
            </button>
          </div>
        ))}
        {!isLoading && clusters.length === 0 && (
          <p className="text-sm text-gray-500 font-mono">No clusters registered.</p>
        )}
      </div>
    </div>
  )
}

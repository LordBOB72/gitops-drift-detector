import type { Cluster, DriftEvent, AuditEntry, ReconcileRequest } from '../types'

const BASE = '/api/v1'

async function req<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...opts?.headers },
    ...opts,
  })
  if (!res.ok) {
    const body = await res.text().catch(() => res.statusText)
    throw new Error(`${res.status}: ${body}`)
  }
  // 204 no content
  if (res.status === 204) return undefined as unknown as T
  return res.json()
}

export const api = {
  clusters: {
    list: () => req<Cluster[]>('/clusters'),
    add: (name: string, kubeconfig: string) =>
      req<{ id: string }>('/clusters', {
        method: 'POST',
        body: JSON.stringify({ name, kubeconfig }),
      }),
    remove: (id: string) => req<void>(`/clusters/${id}`, { method: 'DELETE' }),
  },
  drift: {
    get: (clusterId: string) => req<DriftEvent[]>(`/clusters/${clusterId}/drift`),
    trigger: (clusterId: string) =>
      req<{ status: string }>(`/clusters/${clusterId}/drift/trigger`, { method: 'POST' }),
  },
  reconcile: (clusterId: string, body: ReconcileRequest) =>
    req<{ status: string }>(`/clusters/${clusterId}/reconcile`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  audit: {
    get: (clusterId: string, limit = 100) =>
      req<AuditEntry[]>(`/clusters/${clusterId}/audit?limit=${limit}`),
  },
}

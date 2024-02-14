export type Severity = 'critical' | 'warning' | 'info'
export type DriftType = 'modified' | 'missing' | 'unexpected'

export interface Cluster {
  id: string
  name: string
}

export interface DriftEvent {
  id: string
  cluster_id: string
  resource_kind: string
  resource_name: string
  namespace: string
  severity: Severity
  drift_type: DriftType
  desired_state?: Record<string, unknown>
  live_state?: Record<string, unknown>
  diff?: string
  detected_at: string
}

export interface AuditEntry {
  id: string
  cluster_id: string
  action: string
  actor: string
  resource: string
  detail?: Record<string, unknown>
  created_at: string
}

export interface ReconcileRequest {
  resource_kind: string
  resource_name: string
  namespace: string
  manifest: string
}

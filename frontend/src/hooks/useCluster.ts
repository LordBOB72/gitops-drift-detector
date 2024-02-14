import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'

export function useClusters() {
  return useQuery({ queryKey: ['clusters'], queryFn: api.clusters.list })
}

export function useDrift(clusterId: string | null) {
  return useQuery({
    queryKey: ['drift', clusterId],
    queryFn: () => api.drift.get(clusterId!),
    enabled: !!clusterId,
    refetchInterval: 30_000,
  })
}

export function useAudit(clusterId: string | null) {
  return useQuery({
    queryKey: ['audit', clusterId],
    queryFn: () => api.audit.get(clusterId!),
    enabled: !!clusterId,
    refetchInterval: 60_000,
  })
}

export function useReconcile(clusterId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Parameters<typeof api.reconcile>[1]) =>
      api.reconcile(clusterId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['drift', clusterId] })
      qc.invalidateQueries({ queryKey: ['audit', clusterId] })
    },
  })
}

export function useTriggerDetection(clusterId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.drift.trigger(clusterId),
    onSuccess: () => {
      setTimeout(() => qc.invalidateQueries({ queryKey: ['drift', clusterId] }), 2000)
    },
  })
}

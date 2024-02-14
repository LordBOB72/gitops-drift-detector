import clsx from 'clsx'
import type { Severity } from '../types'

interface BadgeProps {
  severity: Severity
  label?: string
}

const styles: Record<Severity, string> = {
  critical: 'bg-red-500/20 text-red-400 border-red-500/40',
  warning: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/40',
  info: 'bg-blue-500/20 text-blue-400 border-blue-500/40',
}

export function SeverityBadge({ severity, label }: BadgeProps) {
  return (
    <span className={clsx('text-xs font-mono px-2 py-0.5 rounded border', styles[severity])}>
      {label ?? severity}
    </span>
  )
}

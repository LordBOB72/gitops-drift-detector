import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'
import type { DriftEvent } from '../types'
import { format, subDays, startOfDay } from 'date-fns'

interface Props {
  events: DriftEvent[]
}

// bucket events by day for the last 14 days
function buildChartData(events: DriftEvent[]) {
  const days = Array.from({ length: 14 }, (_, i) => {
    const d = startOfDay(subDays(new Date(), 13 - i))
    return { date: d, label: format(d, 'MMM d'), critical: 0, warning: 0, info: 0 }
  })

  for (const ev of events) {
    const evDay = startOfDay(new Date(ev.detected_at)).getTime()
    const bucket = days.find((d) => d.date.getTime() === evDay)
    if (!bucket) continue
    bucket[ev.severity]++
  }

  return days.map(({ label, critical, warning, info }) => ({ label, critical, warning, info }))
}

export function DriftHistoryChart({ events }: Props) {
  const data = buildChartData(events)

  return (
    <ResponsiveContainer width="100%" height={200}>
      <AreaChart data={data} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
        <defs>
          <linearGradient id="critical" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#ef4444" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="warning" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#eab308" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#eab308" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="info" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#1c2a40" />
        <XAxis dataKey="label" tick={{ fill: '#64748b', fontSize: 11 }} axisLine={false} tickLine={false} />
        <YAxis tick={{ fill: '#64748b', fontSize: 11 }} axisLine={false} tickLine={false} allowDecimals={false} />
        <Tooltip
          contentStyle={{ background: '#0f1623', border: '1px solid #1c2a40', borderRadius: 6, fontSize: 12 }}
          labelStyle={{ color: '#94a3b8' }}
        />
        <Legend wrapperStyle={{ fontSize: 12, color: '#64748b' }} />
        <Area type="monotone" dataKey="critical" stroke="#ef4444" fill="url(#critical)" strokeWidth={1.5} />
        <Area type="monotone" dataKey="warning" stroke="#eab308" fill="url(#warning)" strokeWidth={1.5} />
        <Area type="monotone" dataKey="info" stroke="#3b82f6" fill="url(#info)" strokeWidth={1.5} />
      </AreaChart>
    </ResponsiveContainer>
  )
}

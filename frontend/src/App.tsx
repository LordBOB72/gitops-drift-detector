import { useState } from 'react'
import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom'
import { useClusters } from './hooks/useCluster'
import { Dashboard } from './pages/Dashboard'
import { ClustersPage } from './pages/ClustersPage'
import { AuditPage } from './pages/AuditPage'
import { ClusterSwitcher } from './components/ClusterSwitcher'

export default function App() {
  const { data: clusters = [] } = useClusters()
  const [activeCluster, setActiveCluster] = useState<string | null>(null)

  const clusterId = activeCluster ?? clusters[0]?.id ?? null

  return (
    <BrowserRouter>
      <div className="min-h-screen flex flex-col bg-surface-0 font-sans">
        <header className="border-b border-surface-3 bg-surface-1 px-6 py-3 flex items-center gap-6">
          <span className="font-mono text-accent-cyan font-medium tracking-tight text-sm">
            ⎈ drift-detector
          </span>
          <nav className="flex gap-1 ml-4">
            {[
              { to: '/', label: 'Dashboard' },
              { to: '/clusters', label: 'Clusters' },
              { to: '/audit', label: 'Audit Log' },
            ].map(({ to, label }) => (
              <NavLink
                key={to}
                to={to}
                end={to === '/'}
                className={({ isActive }) =>
                  `px-3 py-1.5 rounded text-sm transition-colors ${
                    isActive
                      ? 'bg-surface-3 text-white'
                      : 'text-gray-400 hover:text-white hover:bg-surface-2'
                  }`
                }
              >
                {label}
              </NavLink>
            ))}
          </nav>
          <div className="ml-auto">
            <ClusterSwitcher
              clusters={clusters}
              active={clusterId}
              onChange={setActiveCluster}
            />
          </div>
        </header>

        <main className="flex-1 p-6">
          <Routes>
            <Route path="/" element={<Dashboard clusterId={clusterId} />} />
            <Route path="/clusters" element={<ClustersPage />} />
            <Route path="/audit" element={<AuditPage clusterId={clusterId} />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}

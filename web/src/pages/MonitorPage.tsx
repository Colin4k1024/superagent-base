import { useState } from 'react'
import Header from '../components/Header'
import StatusPanel from '../components/monitor/StatusPanel'
import MetricsPanel from '../components/monitor/MetricsPanel'
import LogsPanel from '../components/monitor/LogsPanel'
import AdminPanel from '../components/monitor/AdminPanel'

type TabId = 'status' | 'metrics' | 'logs' | 'admin'

const TABS: { id: TabId; label: string; icon: string }[] = [
  { id: 'status', label: 'Status', icon: '✦' },
  { id: 'metrics', label: 'Metrics', icon: '◈' },
  { id: 'logs', label: 'Logs', icon: '≡' },
  { id: 'admin', label: 'Admin', icon: '⚙' },
]

export default function MonitorPage() {
  const [activeTab, setActiveTab] = useState<TabId>('status')

  return (
    <div className="flex flex-col h-full">
      <Header title="Monitor" />

      {/* Tab bar */}
      <div className="bg-white border-b border-gray-200 px-6">
        <nav className="flex gap-1 -mb-px" role="tablist">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              role="tab"
              aria-selected={activeTab === tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-1.5 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.id
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <span className="text-base leading-none">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Panel content */}
      <div className="flex-1 overflow-auto p-6">
        {activeTab === 'status' && <StatusPanel />}
        {activeTab === 'metrics' && <MetricsPanel />}
        {activeTab === 'logs' && <LogsPanel />}
        {activeTab === 'admin' && <AdminPanel />}
      </div>
    </div>
  )
}

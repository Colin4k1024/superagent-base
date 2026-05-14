import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Header from '../components/Header'
import StatusPanel from '../components/monitor/StatusPanel'
import MetricsPanel from '../components/monitor/MetricsPanel'
import LogsPanel from '../components/monitor/LogsPanel'
import AdminPanel from '../components/monitor/AdminPanel'

type TabId = 'status' | 'metrics' | 'logs' | 'admin'

const TAB_ICONS: Record<TabId, string> = {
  status: '✦',
  metrics: '◈',
  logs: '≡',
  admin: '⚙',
}

const TAB_IDS: TabId[] = ['status', 'metrics', 'logs', 'admin']

export default function MonitorPage() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<TabId>('status')

  return (
    <div className="flex flex-col h-full">
      <Header title={t('nav.monitor')} />

      {/* Tab bar */}
      <div className="bg-white border-b border-gray-200 px-6">
        <nav className="flex gap-1 -mb-px" role="tablist">
          {TAB_IDS.map((id) => (
            <button
              key={id}
              role="tab"
              aria-selected={activeTab === id}
              onClick={() => setActiveTab(id)}
              className={`flex items-center gap-1.5 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === id
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <span className="text-base leading-none">{TAB_ICONS[id]}</span>
              {t(`monitor.${id}`)}
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

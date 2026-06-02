import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import OverviewTab from '@/components/observability/OverviewTab'
import TracesTab from '@/components/observability/TracesTab'
import TokenUsageTab from '@/components/observability/TokenUsageTab'
import ToolsTab from '@/components/observability/ToolsTab'
import AgentAnalyticsTab from '@/components/observability/AgentAnalyticsTab'
import ModelRoutingTab from '@/components/observability/ModelRoutingTab'

const tabs = [
  { key: 'overview', labelKey: 'observability.tabs.overview' },
  { key: 'traces', labelKey: 'observability.tabs.traces' },
  { key: 'tokens', labelKey: 'observability.tabs.tokens' },
  { key: 'tools', labelKey: 'observability.tabs.tools' },
  { key: 'agents', labelKey: 'observability.tabs.agents' },
  { key: 'routing', labelKey: 'observability.tabs.models' },
] as const

type TabKey = (typeof tabs)[number]['key']

export default function ObservabilityPage() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<TabKey>('overview')

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
        <h1 className="text-xl font-semibold text-gray-900">{t('observability.title')}</h1>
      </div>

      <div className="border-b border-gray-200 px-6">
        <nav className="flex gap-6 -mb-px">
          {tabs.map(({ key, labelKey }) => (
            <button
              key={key}
              onClick={() => setActiveTab(key)}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === key
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {t(labelKey)}
            </button>
          ))}
        </nav>
      </div>

      <div className="flex-1 overflow-auto p-6">
        {activeTab === 'overview' && <OverviewTab />}
        {activeTab === 'traces' && <TracesTab />}
        {activeTab === 'tokens' && <TokenUsageTab />}
        {activeTab === 'tools' && <ToolsTab />}
        {activeTab === 'agents' && <AgentAnalyticsTab />}
        {activeTab === 'routing' && <ModelRoutingTab />}
      </div>
    </div>
  )
}

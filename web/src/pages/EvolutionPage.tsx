import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import Header from '../components/Header'
import { evolutionApi, type EvolutionStats, type GeneItem } from '../lib/api'

export default function EvolutionPage() {
  const { t } = useTranslation()
  const [stats, setStats] = useState<EvolutionStats | null>(null)
  const [genes, setGenes] = useState<GeneItem[]>([])
  const [total, setTotal] = useState(0)
  const [query, setQuery] = useState('')
  const [minConf, setMinConf] = useState(0.0)
  const [genesLoading, setGenesLoading] = useState(false)
  const [genesError, setGenesError] = useState<string | null>(null)
  const [statsError, setStatsError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'overview' | 'genes' | 'federation'>('overview')
  const [fedQuery, setFedQuery] = useState('')
  const [fedResults, setFedResults] = useState<unknown[]>([])
  const [fedLoading, setFedLoading] = useState(false)
  const [fedError, setFedError] = useState<string | null>(null)

  const loadStats = useCallback(async () => {
    setStatsError(null)
    try {
      const s = await evolutionApi.getStats()
      setStats(s)
    } catch (e) {
      setStatsError(e instanceof Error ? e.message : String(e))
    }
  }, [])

  const loadGenes = useCallback(async () => {
    setGenesLoading(true)
    setGenesError(null)
    try {
      const res = await evolutionApi.listGenes({ q: query || undefined, min_confidence: minConf, limit: 50 })
      setGenes(res.genes)
      setTotal(res.total)
    } catch (e) {
      setGenesError(e instanceof Error ? e.message : String(e))
    } finally {
      setGenesLoading(false)
    }
  }, [query, minConf])

  useEffect(() => {
    loadStats()
  }, [loadStats])

  useEffect(() => {
    if (activeTab === 'genes') {
      loadGenes()
    }
    // Only trigger on tab switch, not on query/minConf changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab])

  const handleFederatedSearch = useCallback(async () => {
    if (!fedQuery) return
    setFedLoading(true)
    setFedError(null)
    try {
      const res = await evolutionApi.federatedSearch(fedQuery, 0.5, 10)
      setFedResults(res.results)
    } catch (e) {
      setFedError(e instanceof Error ? e.message : String(e))
    } finally {
      setFedLoading(false)
    }
  }, [fedQuery])

  const tabs = [
    { id: 'overview' as const, label: t('evolution.tabs.overview'), icon: '◈' },
    { id: 'genes' as const, label: t('evolution.tabs.genes'), icon: '🧬' },
    { id: 'federation' as const, label: t('evolution.tabs.federation'), icon: '🌐' },
  ]

  return (
    <div className="flex flex-col h-full">
      <Header title={t('evolution.title')} />

      <div className="bg-white border-b border-gray-200 px-6">
        <nav className="flex gap-1 -mb-px" role="tablist" aria-label={t('evolution.title')}>
          {tabs.map((tab) => (
            <button
              key={tab.id}
              role="tab"
              aria-selected={activeTab === tab.id}
              aria-controls={`tabpanel-${tab.id}`}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-1.5 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.id
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <span className="text-base leading-none" aria-hidden="true">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      <div className="flex-1 overflow-auto p-6">
        {/* Overview tab */}
        {activeTab === 'overview' && (
          <div id="tabpanel-overview" role="tabpanel" className="space-y-6">
            {statsError && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded text-sm" role="alert">
                {statsError}
              </div>
            )}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              <StatCard
                title={t('evolution.status')}
                value={stats?.enabled ? t('evolution.enabled') : t('evolution.disabled')}
                valueClass={stats?.enabled ? 'text-green-600' : 'text-gray-400'}
                icon="⚡"
              />
              <StatCard
                title={t('evolution.experienceRepo')}
                value={stats?.experience_url || '—'}
                icon="🗄"
              />
              <StatCard
                title={t('evolution.nodeId')}
                value={stats?.sender_id || '—'}
                icon="🔖"
              />
              <StatCard
                title={t('evolution.hubUrl')}
                value={stats?.hub_url || t('evolution.hubNotConfigured')}
                icon="🔗"
              />
              <StatCard
                title={t('evolution.peerNodes')}
                value={stats?.peer_nodes !== undefined ? String(stats.peer_nodes) : '—'}
                icon="🌐"
              />
              <StatCard
                title={t('evolution.minConfidence')}
                value={stats?.min_confidence !== undefined ? `${(stats.min_confidence * 100).toFixed(0)}%` : '—'}
                icon="🎯"
              />
            </div>

            {stats && !stats.enabled && (
              <div className="p-4 bg-yellow-50 border border-yellow-200 rounded text-sm text-yellow-800">
                <strong>{t('evolution.disabledHint')}</strong>
              </div>
            )}
          </div>
        )}

        {/* Gene Library tab */}
        {activeTab === 'genes' && (
          <div id="tabpanel-genes" role="tabpanel" className="space-y-4">
            {genesError && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded text-sm" role="alert">
                {genesError}
              </div>
            )}
            <div className="flex gap-3 items-center flex-wrap">
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && loadGenes()}
                placeholder={t('evolution.searchGenes')}
                className="flex-1 min-w-48 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label={t('evolution.searchGenes')}
              />
              <div className="flex items-center gap-2 text-sm text-gray-600">
                <label htmlFor="min-conf">{t('evolution.minConfLabel')}</label>
                <input
                  id="min-conf"
                  type="number"
                  min={0}
                  max={1}
                  step={0.1}
                  value={minConf}
                  onChange={(e) => setMinConf(Number(e.target.value))}
                  className="w-20 px-2 py-1.5 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <button
                onClick={loadGenes}
                disabled={genesLoading}
                className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {genesLoading ? t('evolution.searching') : t('evolution.search')}
              </button>
            </div>

            <div className="text-sm text-gray-500">{t('evolution.genesFound', { count: total })}</div>

            {genes.length === 0 && !genesLoading ? (
              <div className="text-center py-16 text-gray-400 text-sm">
                {stats?.enabled
                  ? t('evolution.noGenesEnabled')
                  : t('evolution.noGenesDisabled')}
              </div>
            ) : (
              <div className="space-y-2">
                {genes.map((gene) => (
                  <GeneCard key={gene.id} gene={gene} />
                ))}
              </div>
            )}
          </div>
        )}

        {/* Federation tab */}
        {activeTab === 'federation' && (
          <div id="tabpanel-federation" role="tabpanel" className="space-y-4">
            {fedError && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded text-sm" role="alert">
                {fedError}
              </div>
            )}
            <div className="flex gap-3">
              <input
                type="text"
                value={fedQuery}
                onChange={(e) => setFedQuery(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleFederatedSearch()}
                placeholder={t('evolution.federatedPlaceholder')}
                className="flex-1 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label={t('evolution.federatedPlaceholder')}
              />
              <button
                onClick={handleFederatedSearch}
                disabled={fedLoading || !fedQuery}
                className="px-4 py-2 bg-indigo-600 text-white text-sm rounded hover:bg-indigo-700 disabled:opacity-50 transition-colors"
              >
                {fedLoading ? t('evolution.searching') : t('evolution.searchNetwork')}
              </button>
            </div>

            {!stats?.hub_url && (
              <div className="p-4 bg-gray-50 border border-gray-200 rounded text-sm text-gray-600">
                {t('evolution.hubNotConfiguredHint')}
              </div>
            )}

            {fedResults.length > 0 && (
              <div className="space-y-2">
                <div className="text-sm text-gray-500">{t('evolution.fedResults', { count: fedResults.length })}</div>
                {fedResults.map((r, i) => (
                  <div key={i} className="p-3 bg-white border border-gray-200 rounded text-sm font-mono text-gray-700 overflow-auto">
                    <pre className="whitespace-pre-wrap break-words">{JSON.stringify(r, null, 2)}</pre>
                  </div>
                ))}
              </div>
            )}

            {fedResults.length === 0 && !fedLoading && fedQuery && (
              <div className="text-center py-12 text-gray-400 text-sm">{t('evolution.noFedResults')}</div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function StatCard({ title, value, icon, valueClass = 'text-gray-800' }: {
  title: string
  value: string
  icon: string
  valueClass?: string
}) {
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4">
      <div className="flex items-center gap-2 mb-2">
        <span className="text-lg" aria-hidden="true">{icon}</span>
        <span className="text-xs font-medium text-gray-500 uppercase tracking-wide">{title}</span>
      </div>
      <div className={`text-sm font-medium break-all ${valueClass}`}>{value}</div>
    </div>
  )
}

function GeneCard({ gene }: { gene: GeneItem }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const successPct = gene.success_rate !== undefined
    ? `${(gene.success_rate * 100).toFixed(0)}%`
    : '—'
  const confPct = `${(gene.confidence * 100).toFixed(0)}%`

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4 hover:border-blue-300 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-xs font-mono text-gray-400 truncate">{gene.id}</span>
            <ConfBadge value={gene.confidence} />
          </div>
          <div className="mt-1 text-sm text-gray-700 line-clamp-2">
            {typeof gene.strategy === 'string'
              ? gene.strategy
              : JSON.stringify(gene.strategy)}
          </div>
        </div>
        <div className="flex flex-col items-end gap-1 shrink-0 text-xs text-gray-500">
          <span>{t('evolution.useCount')}: <strong className="text-gray-700">{gene.use_count}</strong></span>
          <span>{t('evolution.successRate')}: <strong className="text-green-600">{successPct}</strong></span>
          <button
            onClick={() => setExpanded((v) => !v)}
            className="mt-1 text-blue-500 hover:text-blue-700"
            aria-expanded={expanded}
          >
            {expanded ? t('evolution.collapse') : t('evolution.expand')}
          </button>
        </div>
      </div>
      {expanded && (
        <div className="mt-3 p-3 bg-gray-50 rounded text-xs font-mono text-gray-600 overflow-auto">
          <pre className="whitespace-pre-wrap break-words">{JSON.stringify(gene.strategy, null, 2)}</pre>
          <div className="mt-2 space-y-0.5 text-gray-400">
            <div>{t('evolution.confidence')}: {confPct}</div>
            {gene.contributor_id && <div>{t('evolution.contributor')}: {gene.contributor_id}</div>}
            {gene.created_at && <div>{t('evolution.created')}: {gene.created_at}</div>}
          </div>
        </div>
      )}
    </div>
  )
}

function ConfBadge({ value }: { value: number }) {
  const pct = value * 100
  let cls = 'bg-red-100 text-red-700'
  if (pct >= 70) cls = 'bg-green-100 text-green-700'
  else if (pct >= 50) cls = 'bg-yellow-100 text-yellow-700'
  return (
    <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium ${cls}`}>
      {pct.toFixed(0)}%
    </span>
  )
}

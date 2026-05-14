import { useState, useEffect, useCallback } from 'react'
import Header from '../components/Header'
import { evolutionApi, type EvolutionStats, type GeneItem } from '../lib/api'

export default function EvolutionPage() {
  const [stats, setStats] = useState<EvolutionStats | null>(null)
  const [genes, setGenes] = useState<GeneItem[]>([])
  const [total, setTotal] = useState(0)
  const [query, setQuery] = useState('')
  const [minConf, setMinConf] = useState(0.0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'overview' | 'genes' | 'federation'>('overview')
  const [fedQuery, setFedQuery] = useState('')
  const [fedResults, setFedResults] = useState<unknown[]>([])
  const [fedLoading, setFedLoading] = useState(false)

  const loadStats = useCallback(async () => {
    try {
      const s = await evolutionApi.getStats()
      setStats(s)
    } catch (e) {
      setError(String(e))
    }
  }, [])

  const loadGenes = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await evolutionApi.listGenes({ q: query || undefined, min_confidence: minConf, limit: 50 })
      setGenes(res.genes)
      setTotal(res.total)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }, [query, minConf])

  useEffect(() => {
    loadStats()
  }, [loadStats])

  useEffect(() => {
    if (activeTab === 'genes') {
      loadGenes()
    }
  }, [activeTab, loadGenes])

  async function handleFederatedSearch() {
    if (!fedQuery) return
    setFedLoading(true)
    try {
      const res = await evolutionApi.federatedSearch(fedQuery, 0.5, 10)
      setFedResults(res.results)
    } catch (e) {
      setError(String(e))
    } finally {
      setFedLoading(false)
    }
  }

  const tabs = [
    { id: 'overview' as const, label: 'Overview', icon: '◈' },
    { id: 'genes' as const, label: 'Gene Library', icon: '🧬' },
    { id: 'federation' as const, label: 'Federation', icon: '🌐' },
  ]

  return (
    <div className="flex flex-col h-full">
      <Header title="Evolution" />

      {/* Tab bar */}
      <div className="bg-white border-b border-gray-200 px-6">
        <nav className="flex gap-1 -mb-px" role="tablist">
          {tabs.map((tab) => (
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

      <div className="flex-1 overflow-auto p-6">
        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded text-sm">
            {error}
          </div>
        )}

        {/* Overview tab */}
        {activeTab === 'overview' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              <StatCard
                title="Status"
                value={stats?.enabled ? 'Enabled' : 'Disabled'}
                valueClass={stats?.enabled ? 'text-green-600' : 'text-gray-400'}
                icon="⚡"
              />
              <StatCard
                title="Experience Repo"
                value={stats?.experience_url || '—'}
                icon="🗄"
              />
              <StatCard
                title="Node ID"
                value={stats?.sender_id || '—'}
                icon="🔖"
              />
              <StatCard
                title="Hub URL"
                value={stats?.hub_url || 'Not configured'}
                icon="🔗"
              />
              <StatCard
                title="Peer Nodes"
                value={stats?.peer_nodes !== undefined ? String(stats.peer_nodes) : '—'}
                icon="🌐"
              />
              <StatCard
                title="Min Confidence"
                value={stats?.min_confidence !== undefined ? `${(stats.min_confidence * 100).toFixed(0)}%` : '—'}
                icon="🎯"
              />
            </div>

            {stats && !stats.enabled && (
              <div className="p-4 bg-yellow-50 border border-yellow-200 rounded text-sm text-yellow-800">
                <strong>Evolution is disabled.</strong> Set{' '}
                <code className="bg-yellow-100 px-1 rounded">EVOLUTION_ENABLED=true</code> and configure{' '}
                <code className="bg-yellow-100 px-1 rounded">ORIS_EXPERIENCE_URL</code> to activate experience
                self-evolution.
              </div>
            )}
          </div>
        )}

        {/* Gene Library tab */}
        {activeTab === 'genes' && (
          <div className="space-y-4">
            <div className="flex gap-3 items-center flex-wrap">
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && loadGenes()}
                placeholder="Search genes (e.g. web_search, model_invoke)..."
                className="flex-1 min-w-48 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <div className="flex items-center gap-2 text-sm text-gray-600">
                <label htmlFor="min-conf">Min confidence:</label>
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
                disabled={loading}
                className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {loading ? 'Loading...' : 'Search'}
              </button>
            </div>

            <div className="text-sm text-gray-500">{total} gene(s) found</div>

            {genes.length === 0 && !loading ? (
              <div className="text-center py-16 text-gray-400 text-sm">
                {stats?.enabled
                  ? 'No genes found. Run some agent conversations to accumulate experience.'
                  : 'Evolution is disabled — enable it to start accumulating genes.'}
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
          <div className="space-y-4">
            <div className="flex gap-3">
              <input
                type="text"
                value={fedQuery}
                onChange={(e) => setFedQuery(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleFederatedSearch()}
                placeholder="Federated gene search across peer nodes..."
                className="flex-1 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                onClick={handleFederatedSearch}
                disabled={fedLoading || !fedQuery}
                className="px-4 py-2 bg-indigo-600 text-white text-sm rounded hover:bg-indigo-700 disabled:opacity-50 transition-colors"
              >
                {fedLoading ? 'Searching...' : 'Search Network'}
              </button>
            </div>

            {!stats?.hub_url && (
              <div className="p-4 bg-gray-50 border border-gray-200 rounded text-sm text-gray-600">
                Hub is not configured. Set{' '}
                <code className="bg-gray-100 px-1 rounded">ORIS_HUB_URL</code> to enable federated gene sharing
                across multiple superagent-base nodes.
              </div>
            )}

            {fedResults.length > 0 && (
              <div className="space-y-2">
                <div className="text-sm text-gray-500">{fedResults.length} result(s) from peer nodes</div>
                {fedResults.map((r, i) => (
                  <div key={i} className="p-3 bg-white border border-gray-200 rounded text-sm font-mono text-gray-700 overflow-auto">
                    <pre className="whitespace-pre-wrap break-words">{JSON.stringify(r, null, 2)}</pre>
                  </div>
                ))}
              </div>
            )}

            {fedResults.length === 0 && !fedLoading && fedQuery && (
              <div className="text-center py-12 text-gray-400 text-sm">No federated results found.</div>
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
        <span className="text-lg">{icon}</span>
        <span className="text-xs font-medium text-gray-500 uppercase tracking-wide">{title}</span>
      </div>
      <div className={`text-sm font-medium break-all ${valueClass}`}>{value}</div>
    </div>
  )
}

function GeneCard({ gene }: { gene: GeneItem }) {
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
          <span>Use: <strong className="text-gray-700">{gene.use_count}</strong></span>
          <span>Success: <strong className="text-green-600">{successPct}</strong></span>
          <button
            onClick={() => setExpanded((v) => !v)}
            className="mt-1 text-blue-500 hover:text-blue-700"
          >
            {expanded ? 'collapse' : 'expand'}
          </button>
        </div>
      </div>
      {expanded && (
        <div className="mt-3 p-3 bg-gray-50 rounded text-xs font-mono text-gray-600 overflow-auto">
          <pre className="whitespace-pre-wrap break-words">{JSON.stringify(gene.strategy, null, 2)}</pre>
          <div className="mt-2 space-y-0.5 text-gray-400">
            <div>Confidence: {confPct}</div>
            {gene.contributor_id && <div>Contributor: {gene.contributor_id}</div>}
            {gene.created_at && <div>Created: {gene.created_at}</div>}
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

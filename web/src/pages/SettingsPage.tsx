import { useState } from 'react'
import Header from '../components/Header'

const placeholderModels = [
  { provider: 'Anthropic', model: 'claude-opus-4-6', active: true },
  { provider: 'OpenAI', model: 'gpt-4o', active: false },
  { provider: 'Local (Ollama)', model: 'llama3.2', active: false },
]

const placeholderMCPServers = [
  { id: '1', name: 'filesystem', url: 'stdio://npx/-y/@modelcontextprotocol/server-filesystem', status: 'connected' },
  { id: '2', name: 'brave-search', url: 'stdio://npx/-y/@modelcontextprotocol/server-brave-search', status: 'disconnected' },
]

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<'models' | 'mcp'>('models')

  return (
    <div className="flex flex-col h-full">
      <Header title="Settings" />

      <div className="flex-1 overflow-auto p-6">
        {/* Tabs */}
        <div className="flex gap-4 border-b border-gray-200 mb-6">
          {(['models', 'mcp'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`pb-2 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab === 'models' ? 'Model Config' : 'MCP Servers'}
            </button>
          ))}
        </div>

        {activeTab === 'models' && (
          <div className="space-y-4 max-w-2xl">
            <p className="text-sm text-gray-500">
              Configure LLM providers and models for the Model Router.
            </p>
            <div className="bg-white rounded-lg border border-gray-200 divide-y divide-gray-100">
              {placeholderModels.map((m) => (
                <div key={m.model} className="flex items-center justify-between px-4 py-3">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{m.model}</p>
                    <p className="text-xs text-gray-500">{m.provider}</p>
                  </div>
                  <div className="flex items-center gap-3">
                    {m.active && (
                      <span className="text-xs bg-blue-50 text-blue-700 px-2 py-0.5 rounded-full">
                        Active
                      </span>
                    )}
                    <button className="text-xs text-gray-500 hover:text-gray-700">Configure</button>
                  </div>
                </div>
              ))}
            </div>
            <button className="text-sm text-blue-600 hover:underline">+ Add model</button>
          </div>
        )}

        {activeTab === 'mcp' && (
          <div className="space-y-4 max-w-2xl">
            <p className="text-sm text-gray-500">
              Connect Model Context Protocol servers to extend agent capabilities.
            </p>
            <div className="bg-white rounded-lg border border-gray-200 divide-y divide-gray-100">
              {placeholderMCPServers.map((srv) => (
                <div key={srv.id} className="flex items-center justify-between px-4 py-3">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{srv.name}</p>
                    <p className="text-xs font-mono text-gray-400 truncate max-w-xs">{srv.url}</p>
                  </div>
                  <div className="flex items-center gap-3">
                    <span
                      className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                        srv.status === 'connected'
                          ? 'bg-green-100 text-green-700'
                          : 'bg-gray-100 text-gray-500'
                      }`}
                    >
                      {srv.status}
                    </span>
                    <button className="text-xs text-red-600 hover:underline">Remove</button>
                  </div>
                </div>
              ))}
            </div>
            <button className="text-sm text-blue-600 hover:underline">+ Add MCP server</button>
          </div>
        )}
      </div>
    </div>
  )
}

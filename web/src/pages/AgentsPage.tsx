import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { agentsApi, type Agent } from '../lib/api'
import Header from '../components/Header'

export default function AgentsPage() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  useEffect(() => {
    agentsApi
      .list()
      .then((list) => {
        setAgents(list)
        setLoading(false)
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : String(err))
        setLoading(false)
      })
  }, [])

  function handleChat(agentName: string) {
    navigate(`/chat?agent=${encodeURIComponent(agentName)}`)
  }

  return (
    <div className="flex flex-col h-full">
      <Header title="Agents" />

      <div className="flex-1 overflow-auto p-6">
        {loading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <span className="w-2 h-2 rounded-full bg-gray-400 animate-pulse" />
            Loading agents…
          </div>
        )}

        {error && (
          <div className="rounded-md bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
            Failed to load agents: {error}
          </div>
        )}

        {!loading && !error && agents.length === 0 && (
          <div className="text-sm text-gray-500">No agents found.</div>
        )}

        {!loading && agents.length > 0 && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {agents.map((agent) => (
              <div
                key={agent.name}
                className="bg-white rounded-lg border border-gray-200 p-4 flex flex-col gap-3 hover:shadow-sm transition-shadow"
              >
                <div>
                  <h3 className="font-medium text-gray-900 text-sm">{agent.name}</h3>
                  {agent.description && (
                    <p className="mt-1 text-xs text-gray-500 leading-relaxed line-clamp-3">
                      {agent.description}
                    </p>
                  )}
                </div>
                <div className="mt-auto">
                  <button
                    className="px-3 py-1.5 text-xs font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 transition-colors"
                    onClick={() => handleChat(agent.name)}
                  >
                    Chat
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

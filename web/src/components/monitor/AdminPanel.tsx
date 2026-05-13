import { useState } from 'react'
import { adminApi, AdminAgent, ReloadResult } from '../../lib/api'

function AgentTypeTag({ type }: { type: string }) {
  const base = 'inline-block px-2 py-0.5 rounded text-xs font-medium'
  const t = type?.toLowerCase() ?? ''
  if (t.includes('supervisor') || t.includes('orchestrat')) {
    return <span className={`${base} bg-purple-100 text-purple-700`}>{type}</span>
  }
  if (t.includes('chat') || t.includes('model')) {
    return <span className={`${base} bg-blue-100 text-blue-700`}>{type}</span>
  }
  if (t.includes('workflow') || t.includes('plan')) {
    return <span className={`${base} bg-orange-100 text-orange-700`}>{type}</span>
  }
  if (t.includes('parallel') || t.includes('sequential')) {
    return <span className={`${base} bg-teal-100 text-teal-700`}>{type}</span>
  }
  return <span className={`${base} bg-gray-100 text-gray-600`}>{type}</span>
}

interface ReloadState {
  status: 'idle' | 'loading' | 'success' | 'error'
  result: ReloadResult | null
  errorMsg: string | null
}

export default function AdminPanel() {
  const [reload, setReload] = useState<ReloadState>({
    status: 'idle',
    result: null,
    errorMsg: null,
  })

  async function handleReload() {
    setReload({ status: 'loading', result: null, errorMsg: null })
    try {
      const result = await adminApi.triggerReload()
      setReload({ status: 'success', result, errorMsg: null })
    } catch (err) {
      setReload({
        status: 'error',
        result: null,
        errorMsg: err instanceof Error ? err.message : String(err),
      })
    }
  }

  return (
    <div className="space-y-6">
      {/* Hot-reload section */}
      <div className="bg-white rounded-lg border border-gray-200 p-5">
        <h2 className="text-sm font-semibold text-gray-900 mb-1">Hot Reload</h2>
        <p className="text-sm text-gray-500 mb-4">
          Trigger a re-scan of <code className="font-mono bg-gray-100 px-1 rounded">configs/agents/</code>.
          Running conversations are not interrupted.
        </p>

        <div className="flex items-center gap-3">
          <button
            onClick={handleReload}
            disabled={reload.status === 'loading'}
            className="bg-blue-600 text-white rounded px-4 py-2 text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {reload.status === 'loading' ? '↺ Reloading…' : '↺ Reload Agents'}
          </button>

          {reload.status === 'success' && (
            <span className="text-sm text-green-600 font-medium">
              Reloaded — {reload.result?.agent_count} agents loaded
            </span>
          )}
          {reload.status === 'error' && (
            <span className="text-sm text-red-600">{reload.errorMsg}</span>
          )}
        </div>

        {/* Reload result detail */}
        {reload.status === 'success' && reload.result && reload.result.agents?.length > 0 && (
          <div className="mt-4">
            <p className="text-xs text-gray-500 mb-2">Loaded agents after reload:</p>
            <ReloadAgentList agents={reload.result.agents} />
          </div>
        )}
      </div>

      {/* Endpoint reference */}
      <div className="bg-white rounded-lg border border-gray-200 p-5">
        <h2 className="text-sm font-semibold text-gray-900 mb-3">Admin Endpoints</h2>
        <div className="space-y-2">
          {[
            { method: 'GET', path: '/api/v1/admin/status', desc: 'System status, uptime, agent list' },
            { method: 'POST', path: '/api/v1/admin/reload', desc: 'Hot-reload agent YAML configs' },
            { method: 'GET', path: '/api/v1/admin/logs', desc: 'SSE log stream (event: log)' },
            { method: 'GET', path: '/metrics', desc: 'Prometheus exposition format' },
          ].map(({ method, path, desc }) => (
            <div
              key={path}
              className="flex items-start gap-3 bg-gray-50 rounded px-3 py-2 text-sm"
            >
              <span
                className={`shrink-0 font-mono text-xs font-bold px-1.5 py-0.5 rounded ${
                  method === 'GET'
                    ? 'bg-green-100 text-green-700'
                    : 'bg-blue-100 text-blue-700'
                }`}
              >
                {method}
              </span>
              <code className="font-mono text-gray-800 shrink-0">{path}</code>
              <span className="text-gray-500 text-xs">{desc}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Schema reference */}
      <div className="bg-white rounded-lg border border-gray-200 p-5">
        <h2 className="text-sm font-semibold text-gray-900 mb-3">Agent YAML Schema Reference</h2>
        <pre className="bg-gray-50 rounded p-3 text-xs font-mono text-gray-700 overflow-auto whitespace-pre">
{`apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-agent
spec:
  type: chat_model_agent       # chat_model_agent | deep_agent | supervisor
  model:                       # optional — uses default if omitted
    provider: openai
    model_id: gpt-4o
  system_prompt: |
    You are a helpful assistant.
  tools:
    - builtin/web_search       # URI: builtin/<name>
    - mcp://server/tool        # URI: mcp://<server>/<tool>
  interrupt:
    enabled: false
  description: "A brief description"`}
        </pre>
      </div>
    </div>
  )
}

function ReloadAgentList({ agents }: { agents: AdminAgent[] }) {
  return (
    <div className="rounded border border-gray-200 overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="bg-gray-50 text-left text-xs text-gray-500">
            <th className="px-3 py-1.5 font-medium">Name</th>
            <th className="px-3 py-1.5 font-medium">Type</th>
            <th className="px-3 py-1.5 font-medium">Description</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {agents.map((a) => (
            <tr key={a.name}>
              <td className="px-3 py-1.5 font-mono text-gray-900">{a.name}</td>
              <td className="px-3 py-1.5">
                <AgentTypeTag type={a.type} />
              </td>
              <td className="px-3 py-1.5 text-gray-500 text-xs">{a.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

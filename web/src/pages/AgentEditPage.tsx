import { useState, useEffect, useRef } from 'react'
import { useNavigate, useParams, useLocation } from 'react-router-dom'
import { toast } from 'sonner'
import { agentAdminApi } from '../lib/api'
import { Button } from '../components/ui/button'
import { AgentYAMLEditor } from '../components/agent/AgentYAMLEditor'
import { AgentForm } from '../components/agent/AgentForm'

const NEW_AGENT_TEMPLATE = `apiVersion: superagent/v1
kind: Agent
metadata:
  name: my-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: ""
  system_prompt: ""
  tools: []
`

export default function AgentEditPage() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const isNew = !name

  // Support pre-filled YAML from duplicate action
  const initialYaml = (location.state as { yaml?: string } | null)?.yaml ?? NEW_AGENT_TEMPLATE
  const [yamlContent, setYamlContent] = useState(initialYaml)
  const [loading, setLoading] = useState(!isNew)
  const [saving, setSaving] = useState(false)
  const [validating, setValidating] = useState(false)
  // Prevent circular editor ↔ form updates
  const suppressFormUpdate = useRef(false)

  useEffect(() => {
    if (!isNew && name) {
      setLoading(true)
      agentAdminApi
        .get(name)
        .then((data) => {
          setYamlContent(data.yaml)
        })
        .catch((err: unknown) => {
          toast.error(`Failed to load agent: ${err instanceof Error ? err.message : String(err)}`)
        })
        .finally(() => setLoading(false))
    }
  }, [name, isNew])

  function handleEditorChange(value: string) {
    suppressFormUpdate.current = true
    setYamlContent(value)
    // Allow form to re-read YAML after a short debounce
    setTimeout(() => {
      suppressFormUpdate.current = false
    }, 50)
  }

  function handleFormChange(updatedYaml: string) {
    if (suppressFormUpdate.current) return
    setYamlContent(updatedYaml)
  }

  async function handleSave() {
    setSaving(true)
    try {
      if (isNew) {
        await agentAdminApi.create(yamlContent)
        toast.success('Agent created successfully')
        navigate('/agents')
      } else {
        await agentAdminApi.update(name!, yamlContent)
        toast.success('Agent saved successfully')
        navigate('/agents')
      }
    } catch (err) {
      toast.error(`Save failed: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setSaving(false)
    }
  }

  async function handleValidate() {
    setValidating(true)
    try {
      const result = await agentAdminApi.validate(yamlContent)
      if (result.valid) {
        toast.success('YAML is valid')
      } else {
        toast.error(`Validation failed: ${result.error ?? 'Unknown error'}`)
      }
    } catch (err) {
      toast.error(`Validation error: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setValidating(false)
    }
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="flex items-center gap-2 text-sm text-gray-500">
          <span className="w-2 h-2 rounded-full bg-gray-400 animate-pulse" />
          Loading agent…
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Top bar */}
      <header className="shrink-0 flex items-center gap-3 px-6 py-3 bg-white border-b border-gray-200">
        <button
          onClick={() => navigate('/agents')}
          className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
          </svg>
          Agents
        </button>
        <span className="text-gray-300">/</span>
        <h1 className="text-sm font-semibold text-gray-900">
          {isNew ? 'New Agent' : name}
        </h1>
        {!isNew && (
          <span className="ml-1 inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">
            edit
          </span>
        )}
      </header>

      {/* Two-panel editor */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Monaco YAML editor (60%) */}
        <div className="w-3/5 border-r border-gray-200 overflow-hidden">
          <AgentYAMLEditor value={yamlContent} onChange={handleEditorChange} />
        </div>

        {/* Right: Form panel (40%) */}
        <div className="w-2/5 flex flex-col overflow-hidden bg-white">
          <AgentForm yaml={yamlContent} onFormChange={handleFormChange} />
        </div>
      </div>

      {/* Bottom toolbar */}
      <footer className="shrink-0 flex items-center gap-2 px-6 py-3 bg-white border-t border-gray-200">
        <Button onClick={handleSave} loading={saving} size="sm">
          {isNew ? 'Create' : 'Save'}
        </Button>
        <Button variant="outline" size="sm" onClick={handleValidate} loading={validating}>
          Validate
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => toast.info('Test chat coming soon')}
          type="button"
        >
          Test
        </Button>
        <Button variant="ghost" size="sm" onClick={() => navigate('/agents')}>
          Cancel
        </Button>
      </footer>
    </div>
  )
}

import { useState } from 'react'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../ui/tabs'
import { Input } from '../ui/input'
import { Select } from '../ui/select'
import { Button } from '../ui/button'

const AGENT_TYPES = [
  { value: 'chat_model_agent', label: 'Chat Model Agent' },
  { value: 'deep_agent', label: 'Deep Agent' },
  { value: 'supervisor', label: 'Supervisor' },
  { value: 'sequential', label: 'Sequential' },
  { value: 'parallel', label: 'Parallel' },
  { value: 'plan_execute', label: 'Plan & Execute' },
  { value: 'workflow', label: 'Workflow' },
  { value: 'eino_graph', label: 'Eino Graph' },
]

const PROTOCOLS = [
  { value: '', label: '— default —' },
  { value: 'openai', label: 'openai' },
  { value: 'claude', label: 'claude' },
  { value: 'deepseek', label: 'deepseek' },
  { value: 'gemini', label: 'gemini' },
  { value: 'ollama', label: 'ollama' },
]

interface FormState {
  type: string
  system_prompt: string
  model_primary: string
  model_fallback: string
  model_protocol: string
  model_base_url: string
  model_api_key_env: string
  tools: string[]
}

function parseYaml(yaml: string): Partial<FormState> {
  // Minimal line-by-line extractor; avoids a full YAML parser dependency.
  const lines = yaml.split('\n')
  const get = (key: string): string => {
    const line = lines.find((l) => l.trim().startsWith(key + ':'))
    if (!line) return ''
    const val = line.slice(line.indexOf(':') + 1).trim()
    return val.replace(/^["']|["']$/g, '')
  }

  // Tools: collect lines under `tools:` that start with `- `
  const toolsStart = lines.findIndex((l) => l.trim() === 'tools:')
  const tools: string[] = []
  if (toolsStart >= 0) {
    for (let i = toolsStart + 1; i < lines.length; i++) {
      const trimmed = lines[i].trim()
      if (!trimmed.startsWith('- ')) break
      tools.push(trimmed.slice(2).trim().replace(/^["']|["']$/g, ''))
    }
  }

  // System prompt: may be a block scalar (|), grab next indented lines
  const spIdx = lines.findIndex((l) => l.trim().startsWith('system_prompt:'))
  let system_prompt = ''
  if (spIdx >= 0) {
    const spLine = lines[spIdx]
    const val = spLine.slice(spLine.indexOf(':') + 1).trim()
    if (val === '|' || val === '>') {
      // block scalar — collect indented lines
      const indent = spLine.search(/\S/)
      const promptLines: string[] = []
      for (let i = spIdx + 1; i < lines.length; i++) {
        const line = lines[i]
        if (line.trim() === '') { promptLines.push(''); continue }
        if (line.search(/\S/) <= indent) break
        promptLines.push(line.trimStart())
      }
      system_prompt = promptLines.join('\n').trimEnd()
    } else {
      system_prompt = val.replace(/^["']|["']$/g, '')
    }
  }

  return {
    type: get('type'),
    system_prompt,
    model_primary: get('primary'),
    model_fallback: get('fallback'),
    model_protocol: get('protocol'),
    model_base_url: get('base_url'),
    model_api_key_env: get('api_key_env'),
    tools,
  }
}

function serializeToYaml(base: string, state: FormState): string {
  // Re-serialize only the spec section into the existing YAML structure.
  // Strategy: rebuild spec.type, spec.system_prompt, spec.model, spec.tools
  // while keeping metadata untouched.
  const indent = '  '
  const toolLines =
    state.tools.length > 0
      ? `${indent}tools:\n${state.tools.map((t) => `${indent}  - ${t}`).join('\n')}`
      : `${indent}tools: []`

  const modelLines = [
    state.model_primary ? `${indent}  ${indent}primary: ${state.model_primary}` : null,
    state.model_fallback ? `${indent}  ${indent}fallback: ${state.model_fallback}` : null,
    state.model_protocol ? `${indent}  ${indent}protocol: ${state.model_protocol}` : null,
    state.model_base_url ? `${indent}  ${indent}base_url: ${state.model_base_url}` : null,
    state.model_api_key_env ? `${indent}  ${indent}api_key_env: ${state.model_api_key_env}` : null,
  ]
    .filter(Boolean)
    .join('\n')

  const modelSection = modelLines
    ? `${indent}model:\n${modelLines}`
    : `${indent}model:\n${indent}  primary: ""`

  const promptSection = state.system_prompt
    ? `${indent}system_prompt: |\n${state.system_prompt
        .split('\n')
        .map((l) => `${indent}  ${l}`)
        .join('\n')}`
    : `${indent}system_prompt: ""`

  // Keep lines up to and including `spec:`, replace rest
  const specIdx = base.split('\n').findIndex((l) => l.trim() === 'spec:')
  const header =
    specIdx >= 0
      ? base
          .split('\n')
          .slice(0, specIdx + 1)
          .join('\n')
      : base

  const specBody = [
    `${indent}type: ${state.type}`,
    modelSection,
    promptSection,
    toolLines,
  ].join('\n')

  return `${header}\n${specBody}\n`
}

interface AgentFormProps {
  yaml: string
  onFormChange: (updatedYaml: string) => void
}

export function AgentForm({ yaml, onFormChange }: AgentFormProps) {
  const parsed = parseYaml(yaml)
  const [tab, setTab] = useState('basic')
  const [form, setForm] = useState<FormState>({
    type: parsed.type || 'chat_model_agent',
    system_prompt: parsed.system_prompt || '',
    model_primary: parsed.model_primary || '',
    model_fallback: parsed.model_fallback || '',
    model_protocol: parsed.model_protocol || '',
    model_base_url: parsed.model_base_url || '',
    model_api_key_env: parsed.model_api_key_env || '',
    tools: parsed.tools || [],
  })
  const [newTool, setNewTool] = useState('')

  function update(patch: Partial<FormState>) {
    const next = { ...form, ...patch }
    setForm(next)
    onFormChange(serializeToYaml(yaml, next))
  }

  function addTool() {
    const trimmed = newTool.trim()
    if (!trimmed) return
    update({ tools: [...form.tools, trimmed] })
    setNewTool('')
  }

  function removeTool(idx: number) {
    update({ tools: form.tools.filter((_, i) => i !== idx) })
  }

  return (
    <Tabs value={tab} onChange={setTab} className="h-full overflow-hidden flex flex-col">
      <TabsList className="shrink-0 px-4">
        <TabsTrigger value="basic">Basic</TabsTrigger>
        <TabsTrigger value="model">Model</TabsTrigger>
        <TabsTrigger value="tools">Tools</TabsTrigger>
      </TabsList>

      <div className="flex-1 overflow-auto">
        <TabsContent value="basic" className="px-4 space-y-4">
          <Select
            label="Agent Type"
            value={form.type}
            onChange={(e) => update({ type: e.target.value })}
            items={AGENT_TYPES}
          />
          <div className="flex flex-col gap-1">
            <label className="text-sm font-medium text-gray-700">System Prompt</label>
            <textarea
              value={form.system_prompt}
              onChange={(e) => update({ system_prompt: e.target.value })}
              rows={8}
              className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm font-mono
                placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-2
                focus:ring-blue-500/20 resize-y"
              placeholder="You are a helpful assistant…"
            />
          </div>
        </TabsContent>

        <TabsContent value="model" className="px-4 space-y-4">
          <Input
            label="Primary Model"
            value={form.model_primary}
            onChange={(e) => update({ model_primary: e.target.value })}
            placeholder="e.g. gpt-4o"
          />
          <Input
            label="Fallback Model"
            value={form.model_fallback}
            onChange={(e) => update({ model_fallback: e.target.value })}
            placeholder="e.g. gpt-3.5-turbo"
          />
          <Select
            label="Protocol"
            value={form.model_protocol}
            onChange={(e) => update({ model_protocol: e.target.value })}
            items={PROTOCOLS}
          />
          <Input
            label="Base URL"
            value={form.model_base_url}
            onChange={(e) => update({ model_base_url: e.target.value })}
            placeholder="https://api.openai.com/v1"
          />
          <Input
            label="API Key Env Var"
            value={form.model_api_key_env}
            onChange={(e) => update({ model_api_key_env: e.target.value })}
            placeholder="OPENAI_API_KEY"
          />
        </TabsContent>

        <TabsContent value="tools" className="px-4 space-y-3">
          {form.tools.length === 0 && (
            <p className="text-sm text-gray-400">No tools configured.</p>
          )}
          {form.tools.map((tool, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <span className="flex-1 rounded-md border border-gray-200 bg-gray-50 px-3 py-1.5 text-sm font-mono text-gray-700">
                {tool}
              </span>
              <Button
                variant="ghost"
                size="sm"
                type="button"
                onClick={() => removeTool(idx)}
                className="text-red-500 hover:text-red-700 hover:bg-red-50 px-2"
              >
                ✕
              </Button>
            </div>
          ))}
          <div className="flex gap-2 pt-1">
            <input
              value={newTool}
              onChange={(e) => setNewTool(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && addTool()}
              placeholder="builtin/web_search or mcp://server/tool"
              className="flex-1 h-9 rounded-md border border-gray-300 bg-white px-3 py-2 text-sm
                placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-2
                focus:ring-blue-500/20"
            />
            <Button variant="outline" size="sm" type="button" onClick={addTool}>
              Add
            </Button>
          </div>
        </TabsContent>
      </div>
    </Tabs>
  )
}

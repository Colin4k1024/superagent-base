import { Link } from 'react-router-dom'
import { Button } from '../ui/button'
import { cn } from '../../lib/cn'

export interface WorkflowToolbarProps {
  agentName: string
  onSave: () => void
  onExport: () => void
  onAutoLayout: () => void
  saving?: boolean
  dirty?: boolean
}

export function WorkflowToolbar({
  agentName,
  onSave,
  onExport,
  onAutoLayout,
  saving = false,
  dirty = false,
}: WorkflowToolbarProps) {
  return (
    <div
      className={cn(
        'flex items-center justify-between',
        'h-12 px-4 border-b border-gray-200 bg-white shrink-0',
      )}
    >
      {/* Left: back link */}
      <Link
        to="/agents"
        className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-800 transition-colors"
      >
        <svg
          className="h-4 w-4"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.75"
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 4l-6 6 6 6" />
        </svg>
        Back to Agent
      </Link>

      {/* Center: agent name */}
      <span className="absolute left-1/2 -translate-x-1/2 text-sm font-semibold text-gray-900 truncate max-w-xs">
        {agentName}
      </span>

      {/* Right: action buttons */}
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={onAutoLayout}>
          Auto Layout
        </Button>

        <Button variant="outline" size="sm" onClick={onExport}>
          Export YAML
        </Button>

        <Button
          variant="default"
          size="sm"
          onClick={onSave}
          loading={saving}
          disabled={!dirty || saving}
        >
          Save
        </Button>
      </div>
    </div>
  )
}

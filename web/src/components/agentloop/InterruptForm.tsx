import { useState } from 'react'
import type { InterruptField } from '../../lib/agentloop-types'

interface InterruptFormProps {
  reason: string
  fields: InterruptField[]
  onSubmit: (values: Record<string, string>) => void
}

export default function InterruptForm({ reason, fields, onSubmit }: InterruptFormProps) {
  const [values, setValues] = useState<Record<string, string>>({})

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    onSubmit(values)
  }

  return (
    <div className="my-3 p-4 bg-amber-50 border border-amber-200 rounded-lg">
      <div className="flex items-start gap-2 mb-3">
        <span className="text-amber-600 text-lg">⚠️</span>
        <p className="text-sm text-amber-800">{reason}</p>
      </div>
      <form onSubmit={handleSubmit} className="space-y-3">
        {fields.map((f) => (
          <div key={f.name}>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {f.label}
              {f.required && <span className="text-red-500 ml-0.5">*</span>}
            </label>
            {f.type === 'confirm' ? (
              <div className="flex gap-2">
                <button type="submit" className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700">
                  Confirm
                </button>
              </div>
            ) : f.type === 'select' && f.options ? (
              <select
                value={values[f.name] || ''}
                onChange={(e) => setValues((v) => ({ ...v, [f.name]: e.target.value }))}
                className="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm"
              >
                <option value="">Select...</option>
                {f.options.map((opt) => <option key={opt} value={opt}>{opt}</option>)}
              </select>
            ) : (
              <input
                type="text"
                value={values[f.name] || ''}
                onChange={(e) => setValues((v) => ({ ...v, [f.name]: e.target.value }))}
                required={f.required}
                className="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm"
              />
            )}
          </div>
        ))}
        {fields.some((f) => f.type !== 'confirm') && (
          <button type="submit" className="px-4 py-1.5 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700">
            Submit
          </button>
        )}
      </form>
    </div>
  )
}

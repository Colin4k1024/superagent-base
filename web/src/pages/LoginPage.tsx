import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { setApiKey, isAuthenticated } from '@/lib/auth'

const API_BASE = '/api/v1'

export default function LoginPage() {
  const navigate = useNavigate()
  const [apiKey, setApiKeyInput] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  // Already authenticated — skip login
  useEffect(() => {
    if (isAuthenticated()) {
      navigate('/agents', { replace: true })
    }
  }, [navigate])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (apiKey) {
        headers['X-Admin-Key'] = apiKey
      }

      const res = await fetch(`${API_BASE}/admin/status`, { headers })

      if (res.ok) {
        setApiKey(apiKey)
        navigate('/agents', { replace: true })
      } else if (res.status === 401 || res.status === 403) {
        setError('Invalid API key')
      } else {
        setError(`Server error: HTTP ${res.status}`)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-sm bg-white rounded-xl shadow-sm border border-gray-200 p-8">
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-bold text-gray-900">Superagent</h1>
          <p className="mt-1 text-sm text-gray-500">AI Agent Development Platform</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="API Key"
            type="password"
            placeholder="Enter your admin API key"
            value={apiKey}
            onChange={(e) => setApiKeyInput(e.target.value)}
            autoComplete="current-password"
            autoFocus
          />

          {error && (
            <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-md px-3 py-2">
              {error}
            </p>
          )}

          <Button type="submit" className="w-full" loading={loading}>
            Login
          </Button>
        </form>

        <p className="mt-6 text-center text-xs text-gray-400">
          In dev mode, leave the key empty and click Login
        </p>
      </div>
    </div>
  )
}

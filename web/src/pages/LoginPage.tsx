import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { handleLogin, handleRegister, isAuthenticated } from '@/lib/auth'

type Mode = 'login' | 'register'

export default function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  // If already authenticated, redirect immediately
  if (isAuthenticated()) {
    navigate('/agents', { replace: true })
    return null
  }

  const [mode, setMode] = useState<Mode>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    const fn = mode === 'login' ? handleLogin : handleRegister
    const result = await fn({ email, password })

    setLoading(false)

    if (result.status === 'success') {
      navigate('/agents', { replace: true })
    } else {
      setError(result.message)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-sm bg-white rounded-xl shadow-sm border border-gray-200 p-8">
        <h1 className="text-2xl font-bold text-gray-900 text-center mb-1">
          {mode === 'login' ? t('login.title') : t('login.register', 'Register')}
        </h1>
        <p className="text-sm text-gray-500 text-center mb-6">Superagent Base</p>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Email */}
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
              {t('login.email', 'Email')}
            </label>
            <input
              id="email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition-colors"
              placeholder="you@example.com"
            />
          </div>

          {/* Password */}
          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
              {t('login.password', 'Password')}
            </label>
            <input
              id="password"
              type="password"
              required
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition-colors"
              placeholder={mode === 'login' ? '••••••••' : 'At least 6 characters'}
            />
          </div>

          {/* Error */}
          {error && (
            <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-md px-3 py-2">
              {error}
            </p>
          )}

          {/* Submit */}
          <button
            type="submit"
            disabled={loading}
            className="w-full inline-flex items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {loading ? (
              <span className="h-4 w-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : mode === 'login' ? (
              t('login.submit', 'Sign in')
            ) : (
              t('login.registerSubmit', 'Create account')
            )}
          </button>
        </form>

        {/* Toggle mode */}
        <p className="mt-4 text-center text-sm text-gray-500">
          {mode === 'login' ? (
            <>
              {t('login.noAccount', "Don't have an account?")}{' '}
              <button
                type="button"
                onClick={() => { setMode('register'); setError('') }}
                className="text-blue-600 hover:text-blue-700 font-medium"
              >
                {t('login.register', 'Register')}
              </button>
            </>
          ) : (
            <>
              {t('login.hasAccount', 'Already have an account?')}{' '}
              <button
                type="button"
                onClick={() => { setMode('login'); setError('') }}
                className="text-blue-600 hover:text-blue-700 font-medium"
              >
                {t('login.submit', 'Sign in')}
              </button>
            </>
          )}
        </p>
      </div>
    </div>
  )
}

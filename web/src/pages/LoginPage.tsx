import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { isAuthenticated, handleLogin } from '@/lib/auth'
import type { LoginResult } from '@/lib/auth'

export default function LoginPage() {
  const { t } = useTranslation()
  const [error, setError] = useState('')

  function doLogin() {
    setError('')
    handleLogin()
      .then((result: LoginResult) => {
        if (result.status === 'success') {
          window.location.replace('/agents')
        } else if (result.status === 'error') {
          setError(result.message)
        }
        // status === 'redirecting': SDK is navigating to SSO, do nothing
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : String(err))
      })
  }

  useEffect(() => {
    if (isAuthenticated()) {
      window.location.replace('/agents')
      return
    }
    doLogin()
  }, [])

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-sm bg-white rounded-xl shadow-sm border border-gray-200 p-8 text-center">
        <h1 className="text-2xl font-bold text-gray-900 mb-2">{t('login.title')}</h1>
        <p className="text-sm text-gray-500 mb-6">AI Agent Development Platform</p>

        {error ? (
          <div className="space-y-4">
            <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-md px-3 py-2">
              {error}
            </p>
            <button
              onClick={doLogin}
              className="w-full inline-flex items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 transition-colors"
            >
              {t('login.retry')}
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="animate-pulse">
              <div className="mx-auto h-8 w-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
            </div>
            <p className="text-sm text-gray-500">{t('login.redirecting')}</p>
          </div>
        )}
      </div>
    </div>
  )
}

import { configUserCenter } from '@haier/iam'
import './i18n'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { router } from './router'
import { queryClient } from './lib/query-client'
import ErrorBoundary from './components/ErrorBoundary'
import './styles/globals.css'

configUserCenter({
  clientId: import.meta.env.VITE_IAM_CLIENT_ID,
  ssoUrl: import.meta.env.VITE_IAM_SSO_URL,
  tokenUrl: import.meta.env.VITE_IAM_TOKEN_URL,
  redirectUri: `${window.location.origin}/login`,
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
        <Toaster position="top-right" richColors />
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>,
)

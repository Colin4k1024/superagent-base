/**
 * Local auth — uses the built-in passport API instead of @company/iam.
 * Login:    POST /api/passport/web/email/login/
 * Register: POST /api/passport/web/email/register/v2/
 * Logout:   GET  /api/passport/web/logout/
 * Session is stored in an httponly cookie set by the backend.
 */

const SESSION_KEY = 'session_key'

// ---- helpers ----

export function getToken(): string | null {
  return localStorage.getItem(SESSION_KEY) || null
}

export function getAccount(): Record<string, unknown> | null {
  const raw = localStorage.getItem('app-user-info')
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

export function isAuthenticated(): boolean {
  return getToken() !== null
}

export function clearAuth(): void {
  localStorage.removeItem(SESSION_KEY)
  localStorage.removeItem('app-user-info')
}

// ---- API calls ----

export type LoginResult =
  | { status: 'success' }
  | { status: 'error'; message: string }

export async function handleLogin(opts: {
  email: string
  password: string
}): Promise<LoginResult> {
  try {
    const res = await fetch('/api/passport/web/email/login/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: opts.email, password: opts.password }),
    })
    const data = await res.json()
    if (!res.ok) {
      return { status: 'error', message: data?.message || data?.msg || `Login failed (${res.status})` }
    }
    // Mark local session as active — the real session cookie is httponly
    localStorage.setItem(SESSION_KEY, 'active')
    localStorage.setItem('app-user-info', JSON.stringify({ email: opts.email }))
    return { status: 'success' }
  } catch (err) {
    return { status: 'error', message: err instanceof Error ? err.message : String(err) }
  }
}

export async function handleRegister(opts: {
  email: string
  password: string
}): Promise<LoginResult> {
  try {
    const res = await fetch('/api/passport/web/email/register/v2/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: opts.email, password: opts.password }),
    })
    const data = await res.json()
    if (!res.ok) {
      return { status: 'error', message: data?.message || data?.msg || `Register failed (${res.status})` }
    }
    localStorage.setItem(SESSION_KEY, 'active')
    localStorage.setItem('app-user-info', JSON.stringify({ email: opts.email }))
    return { status: 'success' }
  } catch (err) {
    return { status: 'error', message: err instanceof Error ? err.message : String(err) }
  }
}

export async function handleLogout(): Promise<void> {
  try {
    await fetch('/api/passport/web/logout/', { method: 'GET' })
  } catch {
    // best-effort — clear local state either way
  }
  clearAuth()
}

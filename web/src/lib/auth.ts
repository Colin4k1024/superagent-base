import { login as iamLogin } from '@company/iam'

const TOKEN_KEY = 'app-access-token'
const USER_INFO_KEY = 'app-user-info'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY) || null
}

export function getAccount(): Record<string, unknown> | null {
  const raw = localStorage.getItem(USER_INFO_KEY)
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
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_INFO_KEY)
}

export type LoginResult = { status: 'success' } | { status: 'redirecting' } | { status: 'error'; message: string }

export async function handleLogin(options?: { invalidateToken?: boolean }): Promise<LoginResult> {
  const res = await iamLogin(options)
  if (res.success) {
    return { status: 'success' }
  }
  // code 999 means SDK is redirecting to SSO — not an error
  if ('code' in res && res.code === 999) {
    return { status: 'redirecting' }
  }
  return { status: 'error', message: ('errorMessage' in res && res.errorMessage) || 'Login failed' }
}

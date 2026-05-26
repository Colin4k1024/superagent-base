import { login as iamLogin } from '@haier/iam'

const TOKEN_KEY = 'haier-user-center-access-token'
const USER_INFO_KEY = 'haier-user-center-user-info'

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

export async function handleLogin(options?: { invalidateToken?: boolean }): Promise<boolean> {
  const res = await iamLogin(options)
  return res.success
}

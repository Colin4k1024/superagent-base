// Auth state management — stores the admin API key in localStorage.
// The backend accepts an empty key in dev mode (no ADMIN_API_KEY configured).

const AUTH_KEY = 'admin_api_key'

export function getApiKey(): string | null {
  return localStorage.getItem(AUTH_KEY)
}

export function setApiKey(key: string): void {
  localStorage.setItem(AUTH_KEY, key)
}

export function clearApiKey(): void {
  localStorage.removeItem(AUTH_KEY)
}

// Returns true if a key has been stored (including empty string for dev mode).
export function isAuthenticated(): boolean {
  return localStorage.getItem(AUTH_KEY) !== null
}

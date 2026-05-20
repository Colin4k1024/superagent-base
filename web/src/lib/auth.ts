const TOKEN_KEY = 'haier-user-center-access-token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function isAuthenticated(): boolean {
  return !!localStorage.getItem(TOKEN_KEY)
}

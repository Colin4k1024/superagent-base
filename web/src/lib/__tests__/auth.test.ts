import { describe, it, expect, beforeEach, vi } from 'vitest'

vi.mock('@haier/iam', () => ({
  login: vi.fn().mockResolvedValue(undefined),
  logout: vi.fn().mockResolvedValue(undefined),
  configUserCenter: vi.fn(),
}))

import { getToken, getAccount, isAuthenticated, clearAuth } from '../auth'

describe('auth', () => {
  const localStorageMock = {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
  }

  beforeEach(() => {
    vi.stubGlobal('localStorage', localStorageMock)
    vi.clearAllMocks()
  })

  describe('isAuthenticated', () => {
    it('returns false when no token present', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(isAuthenticated()).toBe(false)
    })

    it('returns true when token is present', () => {
      localStorageMock.getItem.mockReturnValue('iam-token-xxx')
      expect(isAuthenticated()).toBe(true)
    })
  })

  describe('getToken', () => {
    it('returns null when no token in localStorage', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(getToken()).toBeNull()
    })

    it('returns token from haier-user-center-access-token key', () => {
      localStorageMock.getItem.mockImplementation((key: string) =>
        key === 'haier-user-center-access-token' ? 'iam-token-123' : null,
      )
      expect(getToken()).toBe('iam-token-123')
    })
  })

  describe('getAccount', () => {
    it('returns null when no account stored', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(getAccount()).toBeNull()
    })

    it('returns parsed account object from haier-user-center-user-info key', () => {
      localStorageMock.getItem.mockImplementation((key: string) =>
        key === 'haier-user-center-user-info'
          ? JSON.stringify({ name: 'test', email: 'test@haier.com' })
          : null,
      )
      const account = getAccount()
      expect(account).toEqual({ name: 'test', email: 'test@haier.com' })
    })
  })

  describe('clearAuth', () => {
    it('removes token and user info keys', () => {
      clearAuth()
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('haier-user-center-access-token')
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('haier-user-center-user-info')
    })
  })
})

import { describe, it, expect, beforeEach, vi } from 'vitest'

// Mock fetch globally
const fetchMock = vi.fn()
vi.stubGlobal('fetch', fetchMock)

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
    fetchMock.mockReset()
  })

  describe('isAuthenticated', () => {
    it('returns false when no token present', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(isAuthenticated()).toBe(false)
    })

    it('returns true when token is present', () => {
      localStorageMock.getItem.mockReturnValue('active')
      expect(isAuthenticated()).toBe(true)
    })
  })

  describe('getToken', () => {
    it('returns null when no token in localStorage', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(getToken()).toBeNull()
    })

    it('returns token from session_key key', () => {
      localStorageMock.getItem.mockImplementation((key: string) =>
        key === 'session_key' ? 'active' : null,
      )
      expect(getToken()).toBe('active')
    })
  })

  describe('getAccount', () => {
    it('returns null when no account stored', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(getAccount()).toBeNull()
    })

    it('returns parsed account object from app-user-info key', () => {
      localStorageMock.getItem.mockImplementation((key: string) =>
        key === 'app-user-info'
          ? JSON.stringify({ email: 'test@example.com' })
          : null,
      )
      const account = getAccount()
      expect(account).toEqual({ email: 'test@example.com' })
    })
  })

  describe('clearAuth', () => {
    it('removes session_key and user info keys', () => {
      clearAuth()
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('session_key')
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('app-user-info')
    })
  })
})

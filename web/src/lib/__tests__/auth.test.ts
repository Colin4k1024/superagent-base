import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getToken, isAuthenticated } from '../auth'

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

  describe('getToken', () => {
    it('returns null when not set', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(getToken()).toBeNull()
      expect(localStorageMock.getItem).toHaveBeenCalledWith('haier-user-center-access-token')
    })

    it('returns stored value', () => {
      localStorageMock.getItem.mockReturnValue('jwt-token-value')
      expect(getToken()).toBe('jwt-token-value')
    })
  })

  describe('isAuthenticated', () => {
    it('returns false when token is null', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(isAuthenticated()).toBe(false)
    })

    it('returns true when token is set', () => {
      localStorageMock.getItem.mockReturnValue('some-token')
      expect(isAuthenticated()).toBe(true)
    })

    it('returns false for empty string', () => {
      localStorageMock.getItem.mockReturnValue('')
      expect(isAuthenticated()).toBe(false)
    })
  })
})

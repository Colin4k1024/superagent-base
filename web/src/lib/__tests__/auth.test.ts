import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getApiKey, setApiKey, clearApiKey, isAuthenticated } from '../auth'

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

  describe('getApiKey', () => {
    it('returns null when not set', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(getApiKey()).toBeNull()
      expect(localStorageMock.getItem).toHaveBeenCalledWith('admin_api_key')
    })

    it('returns stored value', () => {
      localStorageMock.getItem.mockReturnValue('my-secret-key')
      expect(getApiKey()).toBe('my-secret-key')
    })
  })

  describe('setApiKey', () => {
    it('stores value in localStorage', () => {
      setApiKey('test-key')
      expect(localStorageMock.setItem).toHaveBeenCalledWith('admin_api_key', 'test-key')
    })

    it('stores empty string for dev mode', () => {
      setApiKey('')
      expect(localStorageMock.setItem).toHaveBeenCalledWith('admin_api_key', '')
    })
  })

  describe('clearApiKey', () => {
    it('removes key from localStorage', () => {
      clearApiKey()
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('admin_api_key')
    })
  })

  describe('isAuthenticated', () => {
    it('returns false when key is null', () => {
      localStorageMock.getItem.mockReturnValue(null)
      expect(isAuthenticated()).toBe(false)
    })

    it('returns true when key is set', () => {
      localStorageMock.getItem.mockReturnValue('some-key')
      expect(isAuthenticated()).toBe(true)
    })

    it('returns true for empty string key (dev mode)', () => {
      localStorageMock.getItem.mockReturnValue('')
      expect(isAuthenticated()).toBe(true)
    })
  })
})

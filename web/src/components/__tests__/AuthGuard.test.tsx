import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { AuthGuard } from '../AuthGuard'

vi.mock('@/lib/auth', () => ({
  isAuthenticated: vi.fn(),
}))

vi.mock('@/lib/iam', () => ({
  login: vi.fn(),
}))

import { isAuthenticated } from '@/lib/auth'
import { login } from '@/lib/iam'
const mockIsAuthenticated = vi.mocked(isAuthenticated)
const mockLogin = vi.mocked(login)

describe('AuthGuard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('calls login with invalidateToken when not authenticated', () => {
    mockIsAuthenticated.mockReturnValue(false)

    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route element={<AuthGuard />}>
            <Route path="/dashboard" element={<div>Dashboard</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    )

    expect(mockLogin).toHaveBeenCalledWith({ invalidateToken: true })
    expect(screen.queryByText('Dashboard')).not.toBeInTheDocument()
  })

  it('renders children (via Outlet) when authenticated', () => {
    mockIsAuthenticated.mockReturnValue(true)

    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route element={<AuthGuard />}>
            <Route path="/dashboard" element={<div>Dashboard</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    )

    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(mockLogin).not.toHaveBeenCalled()
  })
})

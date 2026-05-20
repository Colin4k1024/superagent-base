import { Outlet } from 'react-router-dom'
import { isAuthenticated } from '@/lib/auth'
import { login } from '@/lib/iam'

export function AuthGuard() {
  if (!isAuthenticated()) {
    login({ invalidateToken: true })
    return null
  }
  return <Outlet />
}

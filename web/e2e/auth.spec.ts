import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test('shows login page when no stored token', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.removeItem('app-access-token')
    })
    await page.goto('/agents')
    // Login page should render with redirecting message
    await expect(page.locator('text=Redirecting to Company Account Center').or(page.locator('text=正在跳转企业账号中心登录'))).toBeVisible({ timeout: 10_000 })
  })

  test('stored token bypasses login', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('app-access-token', 'test-iam-token')
      localStorage.setItem('language', 'en')
    })
    // Mock API to prevent auth errors
    await page.route('**/api/v1/admin/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: [], health: 'healthy' }),
      })
    })
    await page.goto('/agents')
    await page.waitForLoadState('domcontentloaded')
    await page.waitForTimeout(500)
    expect(page.url()).not.toContain('/login')
  })
})

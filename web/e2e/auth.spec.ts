import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test('redirects to login when no stored key', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.removeItem('admin_api_key')
    })
    await page.goto('/agents')
    await expect(page).toHaveURL(/\/login/)
  })

  test('login page renders input and button', async ({ page }) => {
    await page.goto('/login')
    await page.waitForLoadState('domcontentloaded')
    await expect(page.locator('input[type="password"]')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toBeVisible()
  })

  test('dev mode login with empty key', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('language', 'en')
    })
    // Mock the status endpoint that login calls to validate key
    await page.route('**/api/v1/admin/status', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ health: 'healthy', agent_count: 0, ready: true }),
      })
    })
    await page.goto('/login')
    await page.waitForLoadState('domcontentloaded')
    await page.locator('button[type="submit"]').click()
    await expect(page).toHaveURL(/\/agents/, { timeout: 10_000 })
  })

  test('stored key bypasses login', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('admin_api_key', 'test-key')
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

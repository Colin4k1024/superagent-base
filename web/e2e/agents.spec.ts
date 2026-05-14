import { test, expect } from '@playwright/test'

test.describe('Agents Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('language', 'en')
      localStorage.setItem('admin_api_key', '') // dev mode bypass
    })
    await page.goto('/agents')
    // In dev mode with empty key in localStorage, should either show agents or redirect to login
    // If redirected to login, the auth guard allows empty key
    if (page.url().includes('/login')) {
      await page.goto('/agents')
    }
    await page.waitForLoadState('networkidle', { timeout: 15_000 })
  })

  test('shows agents page content', async ({ page }) => {
    // Page should have loaded (either with agents or empty state)
    await expect(page.locator('body')).not.toBeEmpty()
    const url = page.url()
    expect(url).toContain('/agents')
  })

  test('sidebar has navigation links', async ({ page }) => {
    // Check sidebar links exist by href
    await expect(page.locator('a[href="/chat"]')).toBeAttached()
    await expect(page.locator('a[href="/monitor"]')).toBeAttached()
    await expect(page.locator('a[href="/skills"]')).toBeAttached()
    await expect(page.locator('a[href="/settings"]')).toBeAttached()
  })

  test('sidebar navigation works', async ({ page }) => {
    await page.locator('a[href="/chat"]').click()
    await expect(page).toHaveURL(/\/chat/)
  })

  test('navigate to monitor', async ({ page }) => {
    await page.locator('a[href="/monitor"]').click()
    await expect(page).toHaveURL(/\/monitor/)
  })

  test('navigate to skills', async ({ page }) => {
    await page.locator('a[href="/skills"]').click()
    await expect(page).toHaveURL(/\/skills/)
  })
})

import { test, expect } from '@playwright/test'

test.describe('Agents Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('language', 'en')
      localStorage.setItem('admin_api_key', '')
    })
    await page.goto('/agents')
    // Wait for React to render (not networkidle — Vite HMR keeps socket open)
    await page.waitForLoadState('domcontentloaded')
    await page.waitForTimeout(1000)
  })

  test('shows agents page content', async ({ page }) => {
    const url = page.url()
    expect(url).toContain('/agents')
  })

  test('sidebar has navigation links', async ({ page }) => {
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

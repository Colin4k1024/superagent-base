import { test, expect } from '@playwright/test'

test.describe('Agents Page', () => {
  test.beforeEach(async ({ page }) => {
    // Set auth state
    await page.addInitScript(() => {
      localStorage.setItem('language', 'en')
      localStorage.setItem('admin_api_key', 'test-key')
    })
    // Mock admin API to prevent 403 redirects
    await page.route('**/api/v1/admin/agents', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: [
          { name: 'test-agent', type: 'chat_model_agent', description: 'A test agent', status: 'loaded', file: 'test-agent.yaml' }
        ] }),
      })
    })
    await page.route('**/api/v1/admin/status', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ health: 'healthy', agent_count: 1, uptime_seconds: 100, ready: true }),
      })
    })
    await page.goto('/agents')
    await page.waitForLoadState('domcontentloaded')
    await page.waitForTimeout(500)
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

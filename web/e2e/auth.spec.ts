import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Set English locale for consistent test selectors
    await page.addInitScript(() => {
      localStorage.setItem('language', 'en')
    })
  })

  test('redirects to login when not authenticated', async ({ page }) => {
    await page.goto('/agents')
    await expect(page).toHaveURL(/\/login/)
  })

  test('login page has api key input and submit button', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByPlaceholder(/api key|API Key/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /login|登录/i })).toBeVisible()
  })

  test('login page shows platform name', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByText(/Superagent/)).toBeVisible()
  })

  test('can login with empty key in dev mode', async ({ page }) => {
    await page.goto('/login')
    await page.getByRole('button', { name: /login|登录/i }).click()
    await expect(page).toHaveURL(/\/agents/, { timeout: 10_000 })
  })
})

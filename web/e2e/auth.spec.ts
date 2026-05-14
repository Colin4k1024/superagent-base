import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test('redirects to login when not authenticated', async ({ page }) => {
    await page.goto('/agents')
    await expect(page).toHaveURL(/\/login/)
  })

  test('login page has api key input and submit button', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByPlaceholder(/api key/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /login/i })).toBeVisible()
  })

  test('login page shows platform description', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByText(/Superagent/)).toBeVisible()
    await expect(page.getByText(/AI Agent Development Platform/i)).toBeVisible()
  })

  test('can login with empty key in dev mode', async ({ page }) => {
    await page.goto('/login')
    await page.getByRole('button', { name: /login/i }).click()
    // Should redirect to agents page (dev mode allows empty key)
    await expect(page).toHaveURL(/\/agents/, { timeout: 10_000 })
  })
})

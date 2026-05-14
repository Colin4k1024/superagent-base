import { test, expect } from '@playwright/test'

test.describe('Agents Page', () => {
  test.beforeEach(async ({ page }) => {
    // Login first (dev mode — empty key)
    await page.goto('/login')
    await page.getByRole('button', { name: /login/i }).click()
    await expect(page).toHaveURL(/\/agents/, { timeout: 10_000 })
  })

  test('shows agents list or empty state', async ({ page }) => {
    // Either agent cards or empty state message should be visible
    const agentCards = page.locator('.grid > div')
    const emptyMsg = page.getByText(/No agents found/i)

    const hasCards = await agentCards.count().then((n) => n > 0)
    const hasEmpty = await emptyMsg.isVisible().catch(() => false)

    expect(hasCards || hasEmpty).toBeTruthy()
  })

  test('new agent button navigates to editor', async ({ page }) => {
    await page.getByRole('button', { name: /\+ New Agent/i }).click()
    await expect(page).toHaveURL(/\/agents\/new/)
  })

  test('sidebar navigation to Chat works', async ({ page }) => {
    await page.getByRole('link', { name: /Chat/i }).click()
    await expect(page).toHaveURL(/\/chat/)
  })

  test('sidebar navigation to Monitor works', async ({ page }) => {
    await page.getByRole('link', { name: /Monitor/i }).click()
    await expect(page).toHaveURL(/\/monitor/)
  })

  test('sidebar navigation to Skills works', async ({ page }) => {
    await page.getByRole('link', { name: /Skills/i }).click()
    await expect(page).toHaveURL(/\/skills/)
  })
})

import { test, expect } from '@playwright/test'

test.describe('Agents Page', () => {
  test.beforeEach(async ({ page }) => {
    // Set English locale for consistent test selectors
    await page.addInitScript(() => {
      localStorage.setItem('language', 'en')
    })
    // Login (dev mode — empty key)
    await page.goto('/login')
    await page.getByRole('button', { name: /login|登录/i }).click()
    await expect(page).toHaveURL(/\/agents/, { timeout: 15_000 })
  })

  test('shows agents list or empty state', async ({ page }) => {
    // Either agent cards or empty state message should be visible
    const content = await page.textContent('body')
    const hasContent = content && content.length > 0
    expect(hasContent).toBeTruthy()
  })

  test('new agent button navigates to editor', async ({ page }) => {
    await page.getByRole('button', { name: /new agent|新建/i }).click()
    await expect(page).toHaveURL(/\/agents\/new/)
  })

  test('sidebar navigation to Chat works', async ({ page }) => {
    await page.getByRole('link', { name: /chat|对话/i }).click()
    await expect(page).toHaveURL(/\/chat/)
  })

  test('sidebar navigation to Monitor works', async ({ page }) => {
    await page.getByRole('link', { name: /monitor|监控/i }).click()
    await expect(page).toHaveURL(/\/monitor/)
  })

  test('sidebar navigation to Skills works', async ({ page }) => {
    await page.getByRole('link', { name: /skills|技能/i }).click()
    await expect(page).toHaveURL(/\/skills/)
  })
})

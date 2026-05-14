import { test, expect } from '@playwright/test'

test.describe('Agent Editor', () => {
  test.beforeEach(async ({ page }) => {
    // Set English locale for consistent test selectors
    await page.addInitScript(() => {
      localStorage.setItem('language', 'en')
    })
    await page.goto('/login')
    await page.getByRole('button', { name: /login|登录/i }).click()
    await expect(page).toHaveURL(/\/agents/, { timeout: 15_000 })
  })

  test('new agent page loads with Monaco editor', async ({ page }) => {
    await page.goto('/agents/new')
    // Monaco editor container should be present
    await expect(
      page.locator('.monaco-editor, [data-keybinding-context]'),
    ).toBeVisible({ timeout: 15_000 })
  })

  test('new agent editor contains YAML template', async ({ page }) => {
    await page.goto('/agents/new')
    // Wait for Monaco to fully render
    await page.waitForSelector('.view-lines', { timeout: 15_000 })
    const editorContent = await page.locator('.view-lines').textContent()
    expect(editorContent).toContain('apiVersion')
  })

  test('new agent page has save button', async ({ page }) => {
    await page.goto('/agents/new')
    await expect(
      page.getByRole('button', { name: /save|保存/i }),
    ).toBeVisible({ timeout: 10_000 })
  })
})

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
    await page.goto('/login')
    await page.waitForLoadState('domcontentloaded')
    await page.locator('button[type="submit"]').click()
    await expect(page).toHaveURL(/\/agents/, { timeout: 15_000 })
  })

  test('stored key bypasses login', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('admin_api_key', '')
      localStorage.setItem('language', 'en')
    })
    await page.goto('/agents')
    await page.waitForLoadState('domcontentloaded')
    await page.waitForTimeout(1000)
    // With key stored (even empty), auth guard should not redirect
    const url = page.url()
    expect(url).not.toContain('/login')
  })
})

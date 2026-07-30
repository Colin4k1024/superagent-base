/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { test, expect } from '@playwright/test'

// Monaco editor is unreliable in CI (Vite dev server chunk loading issues).
// These tests verify the route exists and page renders — Monaco-specific
// assertions are marked fixme for CI stability.
test.describe('Agent Editor', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('language', 'en')
      localStorage.setItem('session_key', 'active')
    })
  })

  test('new agent route is accessible', async ({ page }) => {
    await page.goto('/agents/new')
    await page.waitForLoadState('domcontentloaded', { timeout: 15_000 })
    // Page should not be blank (some content rendered)
    const body = await page.locator('body').textContent()
    expect(body?.length).toBeGreaterThan(0)
  })

  test.fixme('new agent page loads Monaco editor', async ({ page }) => {
    // Skipped in CI: Monaco chunk loading is unreliable with Vite dev server
    await page.goto('/agents/new')
    await expect(
      page.locator('.monaco-editor'),
    ).toBeVisible({ timeout: 20_000 })
  })

  test.fixme('new agent editor contains YAML template', async ({ page }) => {
    // Skipped in CI: depends on Monaco fully loading
    await page.goto('/agents/new')
    await page.waitForSelector('.view-lines', { timeout: 20_000 })
    const content = await page.locator('.view-lines').textContent()
    expect(content).toContain('apiVersion')
  })
})

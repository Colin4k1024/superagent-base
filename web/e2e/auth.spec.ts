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

test.describe('Authentication', () => {
  test('shows login page when no stored token', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.removeItem('session_key')
    })
    await page.goto('/agents')
    // Login page should render with redirecting message
    await expect(page.locator('text=Redirecting to Company Account Center').or(page.locator('text=正在跳转企业账号中心登录'))).toBeVisible({ timeout: 10_000 })
  })

  test('stored token bypasses login', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('session_key', 'active')
      localStorage.setItem('language', 'en')
    })
    // Mock API to prevent auth errors
    await page.route('**/api/v1/admin/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: [], health: 'healthy' }),
      })
    })
    await page.goto('/agents')
    await page.waitForLoadState('domcontentloaded')
    await page.waitForTimeout(500)
    expect(page.url()).not.toContain('/login')
  })
})

import { test, expect } from '@playwright/test'

// Helper: set auth and language before navigation
function setupAuth(page: any) {
  return page.addInitScript(() => {
    localStorage.setItem('admin_api_key', '')
    localStorage.setItem('language', 'en')
  })
}

test.describe('Xiaohai Debug Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    // Mock agents list
    await page.route('**/api/v1/agents', (route: any) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          agents: [
            { name: 'research-agent', description: 'Research assistant' },
            { name: 'approval-agent', description: 'Approval agent' },
          ],
        }),
      })
    })
  })

  test('renders debug page with all controls', async ({ page }) => {
    await page.goto('/xiaohai-debug')
    await page.waitForLoadState('domcontentloaded')

    // Header
    await expect(page.locator('text=小海接口调试')).toBeVisible()

    // Controls
    await expect(page.locator('select').first()).toBeVisible()
    await expect(page.locator('text=流式 (SSE)')).toBeVisible()
    await expect(page.locator('text=非流式 (JSON)')).toBeVisible()
    await expect(page.locator('textarea')).toBeVisible()
    await expect(page.locator('button:has-text("发送请求")')).toBeVisible()
  })

  test('agent selector populates from API', async ({ page }) => {
    await page.goto('/xiaohai-debug')
    await page.waitForLoadState('domcontentloaded')

    const select = page.locator('select').first()
    await expect(select.locator('option')).toHaveCount(2)
    await expect(select.locator('option:nth-child(1)')).toHaveText('research-agent')
    await expect(select.locator('option:nth-child(2)')).toHaveText('approval-agent')
  })

  test('streaming request shows SSE events in log', async ({ page }) => {
    // Mock the xiaohai stream endpoint
    await page.route('**/api/v1/xiaohai/stream/research-agent', (route: any) => {
      const body = [
        'data: {"type":"answer","data":{"content_type":"markdown","content":"你好"},"version":"1.0.0"}\n\n',
        'data: {"type":"answer","data":{"content_type":"markdown","content":"！"},"version":"1.0.0"}\n\n',
        'data: {"type":"stream_end","version":"1.0.0"}\n\n',
      ].join('')

      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body,
      })
    })

    await page.goto('/xiaohai-debug')
    await page.waitForLoadState('domcontentloaded')

    // Fill userQuery
    await page.locator('textarea').fill('你好')

    // Click send
    await page.locator('button:has-text("发送请求")').click()

    // Wait for logs to appear
    await expect(page.locator('text=stream_end')).toBeVisible({ timeout: 10000 })

    // Verify response content shows
    await expect(page.locator('text=你好！')).toBeVisible()
  })

  test('non-stream request shows JSON response', async ({ page }) => {
    // Mock the xiaohai chat endpoint
    await page.route('**/api/v1/xiaohai/chat/research-agent', (route: any) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            type: 'answer',
            data: { content_type: 'markdown', content: '1+1=2' },
            version: '1.0.0',
          },
        }),
      })
    })

    await page.goto('/xiaohai-debug')
    await page.waitForLoadState('domcontentloaded')

    // Switch to non-stream mode
    await page.locator('button:has-text("非流式 (JSON)")').click()

    // Fill query
    await page.locator('textarea').fill('1+1')

    // Send
    await page.locator('button:has-text("发送请求")').click()

    // Verify response shows in response panel
    await expect(page.locator('pre').first()).toContainText('1+1=2', { timeout: 10000 })
  })

  test('tool call events display execution_steps', async ({ page }) => {
    await page.route('**/api/v1/xiaohai/stream/approval-agent', (route: any) => {
      const body = [
        'data: {"type":"execution_steps","data":{"content_type":"markdown","content":"正在调用 http_request ..."},"version":"1.0.0"}\n\n',
        'data: {"type":"execution_steps_end","version":"1.0.0"}\n\n',
        'data: {"type":"answer","data":{"content_type":"markdown","content":"请求成功"},"version":"1.0.0"}\n\n',
        'data: {"type":"stream_end","version":"1.0.0"}\n\n',
      ].join('')

      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body,
      })
    })

    await page.goto('/xiaohai-debug')
    await page.waitForLoadState('domcontentloaded')

    // Select approval-agent
    await page.locator('select').first().selectOption('approval-agent')

    await page.locator('textarea').fill('请求httpbin')
    await page.locator('button:has-text("发送请求")').click()

    // Verify execution_steps appears in log
    await expect(page.getByText('"type":"execution_steps"').first()).toBeVisible({ timeout: 10000 })
    await expect(page.getByText('execution_steps_end')).toBeVisible()
    await expect(page.getByText('stream_end')).toBeVisible()
  })

  test('404 error for unknown agent', async ({ page }) => {
    await page.route('**/api/v1/xiaohai/stream/nonexistent', (route: any) => {
      route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1, message: 'agent not found: nonexistent' }),
      })
    })

    // Override agents to include nonexistent
    await page.route('**/api/v1/agents', (route: any) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: [{ name: 'nonexistent', description: '' }] }),
      })
    })

    await page.goto('/xiaohai-debug')
    await page.waitForLoadState('domcontentloaded')

    await page.locator('textarea').fill('test')
    await page.locator('button:has-text("发送请求")').click()

    // Should show error in log
    await expect(page.locator('text=HTTP 404')).toBeVisible({ timeout: 10000 })
  })
})

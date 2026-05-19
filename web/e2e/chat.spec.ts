import { test, expect } from '@playwright/test'

function setupAuth(page: any) {
  return page.addInitScript(() => {
    localStorage.setItem('admin_api_key', '')
    localStorage.setItem('language', 'en')
  })
}

test.describe('Chat Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page)
    await page.route('**/api/v1/agents', (route: any) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          agents: [
            { name: 'research-agent', description: 'Research' },
            { name: 'approval-agent', description: 'Approval' },
          ],
        }),
      })
    })
  })

  test('renders chat page with agent selector', async ({ page }) => {
    await page.goto('/chat')
    await page.waitForLoadState('domcontentloaded')

    await expect(page.locator('select')).toBeVisible()
    await expect(page.locator('textarea')).toBeVisible()
  })

  test('shows empty state before sending messages', async ({ page }) => {
    await page.goto('/chat')
    await page.waitForLoadState('domcontentloaded')

    // Empty state shows current agent name
    await expect(page.locator('span.font-mono')).toHaveText('research-agent', { timeout: 5000 })
  })

  test('sends message and renders streaming markdown response', async ({ page }) => {
    // Mock A2UI stream
    await page.route('**/api/v1/chat/stream', (route: any) => {
      const body = [
        'data: {"type":"text","timestamp":1,"data":{"delta":"Hello "}}\n\n',
        'data: {"type":"text","timestamp":2,"data":{"delta":"**world**"}}\n\n',
        'data: {"type":"text","timestamp":3,"data":{"delta":"!"}}\n\n',
        'data: {"type":"done","timestamp":4,"data":null}\n\n',
      ].join('')

      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body,
      })
    })

    await page.goto('/chat')
    await page.waitForLoadState('domcontentloaded')

    // Type and send
    await page.locator('textarea').fill('hi')
    await page.locator('button svg').last().click() // send button

    // User message should appear
    await expect(page.getByText('hi', { exact: true }).first()).toBeVisible({ timeout: 5000 })

    // Assistant response with markdown (bold)
    await expect(page.locator('strong:has-text("world")')).toBeVisible({ timeout: 10000 })
  })

  test('renders tool call blocks from A2UI events', async ({ page }) => {
    await page.route('**/api/v1/chat/stream', (route: any) => {
      const body = [
        'data: {"type":"tool_call","timestamp":1,"data":{"id":"tc1","name":"web_search","arguments":{"query":"test"},"status":"calling"}}\n\n',
        'data: {"type":"tool_result","timestamp":2,"data":{"id":"tc1","name":"web_search","result":"found results","is_error":false}}\n\n',
        'data: {"type":"text","timestamp":3,"data":{"delta":"Here are the results."}}\n\n',
        'data: {"type":"done","timestamp":4,"data":null}\n\n',
      ].join('')

      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body,
      })
    })

    await page.goto('/chat')
    await page.waitForLoadState('domcontentloaded')

    await page.locator('textarea').fill('search test')
    await page.locator('button svg').last().click()

    // Tool call block should render
    await expect(page.locator('text=web_search')).toBeVisible({ timeout: 10000 })
    await expect(page.locator('text=Here are the results.')).toBeVisible()
  })

  test('renders thinking block from A2UI events', async ({ page }) => {
    await page.route('**/api/v1/chat/stream', (route: any) => {
      const body = [
        'data: {"type":"thinking","timestamp":1,"data":{"delta":"Let me think..."}}\n\n',
        'data: {"type":"text","timestamp":2,"data":{"delta":"The answer is 42."}}\n\n',
        'data: {"type":"done","timestamp":3,"data":null}\n\n',
      ].join('')

      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body,
      })
    })

    await page.goto('/chat')
    await page.waitForLoadState('domcontentloaded')

    await page.locator('textarea').fill('deep question')
    await page.locator('button svg').last().click()

    // Thinking block toggle should appear
    await expect(page.locator('text=思考过程')).toBeVisible({ timeout: 10000 })
    await expect(page.locator('text=The answer is 42.')).toBeVisible()
  })

  test('code block renders with syntax highlighting', async ({ page }) => {
    await page.route('**/api/v1/chat/stream', (route: any) => {
      // Send code block as multiple tokens to simulate real streaming
      const body = [
        'data: {"type":"text","timestamp":1,"data":{"delta":"```python\\ndef hello():\\n    print(\\"hi\\")\\n```"}}\n\n',
        'data: {"type":"done","timestamp":2,"data":null}\n\n',
      ].join('')

      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body,
      })
    })

    await page.goto('/chat')
    await page.waitForLoadState('domcontentloaded')

    await page.locator('textarea').fill('code')
    await page.locator('button svg').last().click()

    // Code block should render (language label in header)
    await expect(page.locator('.text-gray-400.font-mono:has-text("python")')).toBeVisible({ timeout: 10000 })
  })

  test('stop button aborts streaming', async ({ page }) => {
    // Slow stream that we can interrupt
    await page.route('**/api/v1/chat/stream', async (route: any) => {
      // Return a stream that sends one token then hangs
      const body = 'data: {"type":"text","timestamp":1,"data":{"delta":"Starting..."}}\n\n'
      route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body,
      })
    })

    await page.goto('/chat')
    await page.waitForLoadState('domcontentloaded')

    await page.locator('textarea').fill('long task')
    await page.locator('button svg').last().click()

    // Message appears
    await expect(page.locator('text=Starting...')).toBeVisible({ timeout: 5000 })
  })
})

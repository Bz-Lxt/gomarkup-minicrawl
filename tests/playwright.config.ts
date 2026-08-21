import { defineConfig, devices } from '@playwright/test'

const admin = process.env.ADMIN_URL || 'http://127.0.0.1:18422'
const vis = process.env.USER_URL || 'http://127.0.0.1:18423'

export default defineConfig({
  testDir: '.',
  timeout: 60_000,
  fullyParallel: false,
  retries: 0,
  use: {
    baseURL: admin,
    trace: 'off',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  metadata: { vis },
})

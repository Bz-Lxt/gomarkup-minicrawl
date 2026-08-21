import { test, expect } from '@playwright/test'

const vis = process.env.USER_URL || 'http://127.0.0.1:18423'

test('admin crawl search and visualization', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('MINICRAWL')).toBeVisible()

  await page.getByTestId('nav-tasks').click()
  await page.getByTestId('btn-start-task').click()
  await expect(page.getByText('completed').first()).toBeVisible({ timeout: 35000 })

  await page.getByTestId('nav-search').click()
  await page.getByTestId('search-input').fill('minicrawl')
  await page.getByTestId('search-submit').click()
  await expect(page.locator('mark').first()).toBeVisible({ timeout: 10000 })

  await page.goto(vis)
  await expect(page.getByText('MINICRAWL SONAR')).toBeVisible()
  await expect(page.locator('canvas')).toBeVisible()
})

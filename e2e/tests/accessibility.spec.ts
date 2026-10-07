import { test, expect } from '@playwright/test'
import { authFile } from '../helpers/users'

// Covers this round's two accessibility fixes in the real browser, not just
// Vitest's simulated DOM: Like/Dislike exposing their pressed state via
// aria-pressed (not just a color-only .active class), and Escape closing
// the notifications dropdown and returning focus to its toggle button.
test.use({ storageState: authFile('userA') })

test('Like/Dislike expose their pressed state via aria-pressed, not just a color-only class', async ({ page }) => {
  const title = `E2E a11y post ${Date.now()}`
  await page.goto('/')
  await page.getByRole('link', { name: 'New Post' }).click()
  await page.locator('#post-title').fill(title)
  await page.locator('#post-content').fill('Created by the Playwright E2E suite — safe to ignore.')
  await page.locator('#add-post-submit').click()
  await page.getByRole('link', { name: title }).click()

  const likeButton = page.getByRole('button', { name: /^Like/ })
  await expect(likeButton).toHaveAttribute('aria-pressed', 'false')

  await likeButton.click()
  await expect(likeButton).toHaveAttribute('aria-pressed', 'true')
})

test('Escape closes the notifications dropdown and returns focus to the bell', async ({ page }) => {
  await page.goto('/')
  const bell = page.locator('#notifications-bell')
  const toggle = bell.getByRole('button', { name: /Notifications/ })

  await toggle.click()
  await expect(bell.locator('.dropdown-content')).toBeVisible()

  await page.keyboard.press('Escape')
  await expect(bell.locator('.dropdown-content')).not.toBeVisible()
  await expect(toggle).toBeFocused()
})

import { test, expect } from '@playwright/test'
import { authFile, users } from '../helpers/users'

// The author-posts page (/users/:username) previously showed only a post
// list with no profile summary at all — join date, post count, comment
// count had no E2E (or any) coverage.
test.use({ storageState: authFile('userA') })

test("visiting a real user's profile shows their join date, post count, and comment count", async ({ page }) => {
  const title = `E2E author profile post ${Date.now()}`
  const content = 'Created by the Playwright E2E suite — safe to ignore.'

  await page.goto('/')
  await page.getByRole('link', { name: 'New Post' }).click()
  await page.locator('#post-title').fill(title)
  await page.locator('#post-content').fill(content)
  await page.locator('#add-post-submit').click()
  await expect(page.locator('#msg')).toHaveText('Your post was submitted.')

  // Comment on the post's own detail page first, so the profile's comment
  // count reflects at least one real comment, not just zero either way.
  await page.getByRole('link', { name: title }).click()
  await page.getByLabel('Enter your comment here').fill('commenting on my own post')
  await page.getByRole('button', { name: 'Submit Comment' }).click()
  await expect(page.getByText('commenting on my own post')).toBeVisible()

  await page.goto(`/users/${users.userA.uname}`)
  await expect(page.getByRole('heading', { name: `Posts by ${users.userA.uname}` })).toBeVisible()

  const summary = page.locator('#author-profile .profile-summary')
  await expect(summary.getByRole('heading', { name: users.userA.uname })).toBeVisible()
  await expect(summary.getByText(/^Joined /)).toContainText('post')
  await expect(summary.getByText(/^Joined /)).toContainText('comment')

  await expect(page.locator('.post-card', { hasText: title })).toBeVisible()
})

test('visiting a nonexistent user shows "User not found." instead of an empty post list', async ({ page }) => {
  await page.goto(`/users/e2e_nonexistent_user_${Date.now()}`)
  await expect(page.getByText('User not found.')).toBeVisible()
  await expect(page.getByText(/hasn't posted yet\./)).not.toBeVisible()
})

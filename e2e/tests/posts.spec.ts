import { test, expect } from '@playwright/test'
import { authFile } from '../helpers/users'

test.use({ storageState: authFile('userA') })

test('create a post, see it in the feed, and view its detail page', async ({ page }) => {
  const title = `E2E test post ${Date.now()}`
  const content = 'Created by the Playwright E2E suite — safe to ignore.'

  await page.goto('/')
  await page.getByRole('link', { name: 'New Post' }).click()

  await page.locator('#post-title').fill(title)
  await page.locator('#post-content').fill(content)
  await page.locator('#add-post-submit').click()

  // AddPostForm navigates back to the feed and shows a success toast on
  // submit — both are worth asserting on, not just the post's eventual
  // appearance in the list.
  await expect(page.locator('#msg')).toHaveText('Your post was submitted.')

  const postLink = page.getByRole('link', { name: title })
  await expect(postLink).toBeVisible()

  await postLink.click()
  await expect(page.getByRole('heading', { name: title })).toBeVisible()
  await expect(page.getByText(content)).toBeVisible()
})

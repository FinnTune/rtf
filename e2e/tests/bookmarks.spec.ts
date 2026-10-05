import { test, expect } from '@playwright/test'
import { authFile } from '../helpers/users'

// Bookmarking a post (save for later, browse them on their own page) had no
// E2E coverage at all. Purely personal/single-user — no live-update or
// multi-tab aspect, unlike reactions' broadcast counts — so this uses the
// simple page-fixture + storageState pattern like posts.spec.ts, not manual
// contexts.
test.use({ storageState: authFile('userA') })

test('bookmarking a post from the feed, finding it on the bookmarks page, and un-bookmarking it there', async ({ page }) => {
  const title = `E2E bookmark test post ${Date.now()}`
  const content = 'Created by the Playwright E2E suite — safe to ignore.'

  await page.goto('/')
  await page.getByRole('link', { name: 'New Post' }).click()
  await page.locator('#post-title').fill(title)
  await page.locator('#post-content').fill(content)
  await page.locator('#add-post-submit').click()
  await expect(page.locator('#msg')).toHaveText('Your post was submitted.')

  // Scoped to this post's own card — the feed can have other posts (from
  // other spec files sharing the same server) with their own Bookmark
  // buttons.
  const postCard = page.locator('.post-card', { hasText: title })
  await expect(postCard.getByRole('button', { name: 'Bookmark' })).toBeVisible()
  await postCard.getByRole('button', { name: 'Bookmark' }).click()
  await expect(postCard.getByRole('button', { name: 'Bookmarked' })).toBeVisible()

  await page.getByRole('link', { name: 'Bookmarks' }).click()
  await expect(page.getByRole('heading', { name: 'Bookmarked Posts' })).toBeVisible()
  const bookmarkedCard = page.locator('.post-card', { hasText: title })
  await expect(bookmarkedCard).toBeVisible()
  await expect(bookmarkedCard.getByRole('button', { name: 'Bookmarked' })).toBeVisible()

  // Un-bookmark it from the bookmarks page itself.
  await bookmarkedCard.getByRole('button', { name: 'Bookmarked' }).click()
  await expect(bookmarkedCard.getByRole('button', { name: 'Bookmark' })).toBeVisible()

  // The list itself doesn't refetch on a local toggle (purely optimistic,
  // client-side) — reloading the page is what proves the un-bookmark
  // actually persisted server-side, not just in this tab's local state.
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Bookmarked Posts' })).toBeVisible()
  await expect(page.locator('.post-card', { hasText: title })).not.toBeVisible()
})

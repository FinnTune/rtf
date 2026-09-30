import { test, expect, type Page } from '@playwright/test'
import { authFile } from '../helpers/users'

// Comments and post reactions have had several live-update/pagination bug
// fixes this project's history, but neither had any E2E coverage at all —
// this file closes that gap, mirroring posts.spec.ts's own
// create-post-then-navigate-to-detail pattern for getting to a post where
// they live.
test.use({ storageState: authFile('userA') })

async function createPostAndOpenDetail(page: Page, title: string) {
  const content = 'Created by the Playwright E2E suite — safe to ignore.'
  await page.goto('/')
  await page.getByRole('link', { name: 'New Post' }).click()
  await page.locator('#post-title').fill(title)
  await page.locator('#post-content').fill(content)
  await page.locator('#add-post-submit').click()
  await page.getByRole('link', { name: title }).click()
  await expect(page.getByRole('heading', { name: title })).toBeVisible()
}

test('add, edit, and delete a comment on a post', async ({ page }) => {
  await createPostAndOpenDetail(page, `E2E comment test post ${Date.now()}`)

  const commentText = `E2E comment ${Date.now()}`
  await page.getByLabel('Enter your comment here').fill(commentText)
  await page.getByRole('button', { name: 'Submit Comment' }).click()
  await expect(page.getByText(commentText)).toBeVisible()

  // Edit it. Deliberately not just commentText + a suffix — a genuinely
  // distinct string, so a later "the old text is gone" check can't
  // trivially pass just because the old text happens to be a substring of
  // the new one.
  //
  // Not filtered by hasText: this post has exactly one comment, and
  // CommentItem's edit mode swaps the comment's rendered content to an
  // <input> — whose value isn't part of the element's text content, so a
  // hasText-filtered locator captured before entering edit mode stops
  // matching once inside it. An unfiltered locator is re-resolved live
  // against the current DOM at each action instead.
  const commentRow = page.locator('.comment')
  await commentRow.getByRole('button', { name: 'Edit' }).click()
  const editedText = `E2E comment EDITED ${Date.now()}`
  await commentRow.getByLabel('Edit comment').fill(editedText)
  await commentRow.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByText(editedText)).toBeVisible()
  await expect(page.getByText(commentText)).not.toBeVisible()

  // Delete it.
  page.once('dialog', (dialog) => void dialog.accept())
  await commentRow.getByRole('button', { name: 'Delete' }).click()
  await expect(page.getByText(editedText)).not.toBeVisible()
})

test('liking and disliking a post updates the counts, and switching between them moves the count rather than just adding to it', async ({
  page,
}) => {
  await createPostAndOpenDetail(page, `E2E reaction test post ${Date.now()}`)

  const likeButton = page.getByRole('button', { name: /^Like/ })
  const dislikeButton = page.getByRole('button', { name: /^Dislike/ })

  await expect(likeButton).toHaveText('Like (0)')
  await expect(dislikeButton).toHaveText('Dislike (0)')

  await likeButton.click()
  await expect(likeButton).toHaveText('Like (1)')
  await expect(dislikeButton).toHaveText('Dislike (0)')

  // Switching straight to dislike should move the count (ReactToPostHandler's
  // state machine), not leave a stale like behind.
  await dislikeButton.click()
  await expect(dislikeButton).toHaveText('Dislike (1)')
  await expect(likeButton).toHaveText('Like (0)')

  // Clicking the active reaction again toggles it off entirely.
  await dislikeButton.click()
  await expect(dislikeButton).toHaveText('Dislike (0)')
  await expect(likeButton).toHaveText('Like (0)')
})

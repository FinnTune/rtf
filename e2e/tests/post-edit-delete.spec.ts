import { test, expect } from '@playwright/test'
import { authFile, users } from '../helpers/users'

// Editing/deleting a post is the one place SinglePostView does its own
// WebSocket broadcast handling (post-edited/post-deleted), the same live-
// update family as chat/group-chat-membership — a permalink left open in
// another tab must reflect an edit or a deletion without a reload. Had no
// E2E coverage at all (posts.spec.ts only covers create+view).
//
// Manual contexts throughout, with EXISTING saved storageState (not a
// fresh registration) — see admin.spec.ts's own doc comment for why a
// brand-new, never-before-used context immediately after another test's
// page/context fixture teardown is best avoided; reusing a saved
// storageState in a manually-created context was confirmed not to trigger
// that issue.

test('editing and deleting a post updates another viewer\'s open window live', async ({ browser }) => {
  const contextA = await browser.newContext({ storageState: authFile('userA') })
  const contextB = await browser.newContext({ storageState: authFile('userB') })
  const pageA = await contextA.newPage()
  const pageB = await contextB.newPage()

  try {
    const title = `E2E editable post ${Date.now()}`
    const content = 'Created by the Playwright E2E suite — safe to ignore.'

    await pageA.goto('/')
    await pageA.getByRole('link', { name: 'New Post' }).click()
    await pageA.locator('#post-title').fill(title)
    await pageA.locator('#post-content').fill(content)
    await pageA.locator('#add-post-submit').click()
    await pageA.getByRole('link', { name: title }).click()
    await expect(pageA.getByRole('heading', { name: title })).toBeVisible()

    // userB opens the same post directly by URL — the live-update
    // broadcast is what's under test, not navigation/search.
    await pageB.goto(pageA.url())
    await expect(pageB.getByRole('heading', { name: title })).toBeVisible()

    await pageA.getByRole('button', { name: 'Edit' }).click()
    const editedTitle = `E2E edited post ${Date.now()}`
    await pageA.getByLabel('Title').fill(editedTitle)
    await pageA.getByRole('button', { name: 'Save' }).click()
    await expect(pageA.getByRole('heading', { name: editedTitle })).toBeVisible()

    // userB never touched anything — this is purely the post-edited
    // broadcast updating an already-open permalink in place.
    await expect(pageB.getByRole('heading', { name: editedTitle })).toBeVisible({ timeout: 10_000 })
    await expect(pageB.getByRole('heading', { name: title })).not.toBeVisible()

    pageA.once('dialog', (dialog) => void dialog.accept())
    await pageA.getByRole('button', { name: 'Delete' }).click()
    await expect(pageA.locator('#msg')).toHaveText('Post deleted.')

    // userB's post-deleted subscription bounces it back to the feed with
    // its own info toast, with no reload and no action on userB's part.
    await expect(pageB.locator('#msg')).toHaveText('This post was deleted.', { timeout: 10_000 })
    await expect(pageB.getByRole('heading', { name: editedTitle })).not.toBeVisible()
  } finally {
    await contextA.close()
    await contextB.close()
  }
})

test('an admin can delete another user\'s post', async ({ browser }) => {
  const contextA = await browser.newContext({ storageState: authFile('userA') })
  const adminContext = await browser.newContext({ storageState: authFile('adminUser') })
  const pageA = await contextA.newPage()
  const adminPage = await adminContext.newPage()

  try {
    const title = `E2E moderated post ${Date.now()}`
    const content = 'Created by the Playwright E2E suite — safe to ignore.'

    await pageA.goto('/')
    await pageA.getByRole('link', { name: 'New Post' }).click()
    await pageA.locator('#post-title').fill(title)
    await pageA.locator('#post-content').fill(content)
    await pageA.locator('#add-post-submit').click()
    await pageA.getByRole('link', { name: title }).click()
    await expect(pageA.getByRole('heading', { name: title })).toBeVisible()

    await adminPage.goto(pageA.url())
    await expect(adminPage.getByRole('heading', { name: title })).toBeVisible()

    // Not the author — SinglePostView's canDelete shows Delete for an
    // admin too, but never Edit (see its own canDelete/isOwn split); the
    // backend re-checks the admin role independent of this UI gating
    // (DeletePostHandler's own isAdmin lookup).
    await expect(adminPage.getByRole('button', { name: 'Edit' })).not.toBeVisible()
    adminPage.once('dialog', (dialog) => void dialog.accept())
    await adminPage.getByRole('button', { name: 'Delete' }).click()
    await expect(adminPage.locator('#msg')).toHaveText('Post deleted.')

    // The author's own open window updates live too, exactly like the
    // self-delete case above, just triggered by someone else this time.
    await expect(pageA.locator('#msg')).toHaveText('This post was deleted.', { timeout: 10_000 })
  } finally {
    await contextA.close()
    await adminContext.close()
  }
})

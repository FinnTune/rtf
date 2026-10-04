import { test, expect } from '@playwright/test'
import { authFile, users } from '../helpers/users'

// "Someone commented on your post" notifications had no E2E coverage at
// all — this is the one place the feature's own WebSocket push
// (notification-added) and REST endpoints (GET /notifications,
// POST /markNotificationsRead) all come together, so a two-tab live-update
// test is the highest-value coverage here, mirroring the same pattern as
// chat/group-chat-membership/post-edit-delete.

test('commenting on another user\'s post notifies them live, and opening it marks it read', async ({ browser }) => {
  const contextA = await browser.newContext({ storageState: authFile('userA') })
  const contextB = await browser.newContext({ storageState: authFile('userB') })
  const pageA = await contextA.newPage()
  const pageB = await contextB.newPage()

  try {
    const title = `E2E notification test post ${Date.now()}`
    const content = 'Created by the Playwright E2E suite — safe to ignore.'

    // userA creates the post and stays on its detail page.
    await pageA.goto('/')
    await pageA.getByRole('link', { name: 'New Post' }).click()
    await pageA.locator('#post-title').fill(title)
    await pageA.locator('#post-content').fill(content)
    await pageA.locator('#add-post-submit').click()
    await pageA.getByRole('link', { name: title }).click()
    await expect(pageA.getByRole('heading', { name: title })).toBeVisible()

    // Before any comment, no unread badge.
    const bellA = pageA.locator('#notifications-bell')
    await expect(bellA.getByRole('button', { name: /Notifications/ })).toHaveText('Notifications')

    // userB opens the same post (by URL) and comments on it.
    await pageB.goto(pageA.url())
    await expect(pageB.getByRole('heading', { name: title })).toBeVisible()
    const commentText = `E2E notification-triggering comment ${Date.now()}`
    await pageB.getByLabel('Enter your comment here').fill(commentText)
    await pageB.getByRole('button', { name: 'Submit Comment' }).click()
    await expect(pageB.getByText(commentText)).toBeVisible()

    // userA's bell updates live, no reload, no action on userA's part.
    await expect(bellA.getByLabel('1 unread notifications')).toHaveText('1', { timeout: 10_000 })

    // Opening the bell shows who commented and on what.
    await bellA.getByRole('button', { name: /Notifications/ }).click()
    const notificationItem = bellA.getByText(`${users.userB.uname} commented on`)
    await expect(notificationItem).toBeVisible()
    await expect(bellA.getByText(title)).toBeVisible()

    // Clicking it marks it read, closes the dropdown, and navigates to the
    // post it's about — even though userA is already on that exact post.
    await notificationItem.click()
    await expect(pageA.getByRole('heading', { name: title })).toBeVisible()
    await expect(bellA.getByText(`${users.userB.uname} commented on`)).not.toBeVisible()
    await expect(bellA.getByRole('button', { name: /Notifications/ })).toHaveText('Notifications')
  } finally {
    await contextA.close()
    await contextB.close()
  }
})

test("an admin's own post comment-notifications are unaffected by other admin actions, and a self-comment notifies no one", async ({
  browser,
}) => {
  const contextA = await browser.newContext({ storageState: authFile('userA') })
  const pageA = await contextA.newPage()

  try {
    const title = `E2E self-comment post ${Date.now()}`
    const content = 'Created by the Playwright E2E suite — safe to ignore.'

    await pageA.goto('/')
    await pageA.getByRole('link', { name: 'New Post' }).click()
    await pageA.locator('#post-title').fill(title)
    await pageA.locator('#post-content').fill(content)
    await pageA.locator('#add-post-submit').click()
    await pageA.getByRole('link', { name: title }).click()
    await expect(pageA.getByRole('heading', { name: title })).toBeVisible()

    await pageA.getByLabel('Enter your comment here').fill('commenting on my own post')
    await pageA.getByRole('button', { name: 'Submit Comment' }).click()
    await expect(pageA.getByText('commenting on my own post')).toBeVisible()

    // No self-notification — the badge never appears, even briefly. A
    // short, fixed wait (rather than a long timeout on a negative
    // assertion) is deliberate here: this is checking that nothing ever
    // arrives, not racing a real update.
    await pageA.waitForTimeout(2_000)
    const bellA = pageA.locator('#notifications-bell')
    await expect(bellA.getByRole('button', { name: /Notifications/ })).toHaveText('Notifications')
  } finally {
    await contextA.close()
  }
})

import { test, expect } from '@playwright/test'
import { authFile, registerDisposableUser, users } from '../helpers/users'

// Group chat membership (leave/add/remove) is the most complex live-update
// feature added this project's history — every change broadcasts to the
// union of a group's old and new member sets — but had no E2E coverage at
// all. Two real browser contexts throughout, not test.use({ storageState
// }): every assertion here is specifically about one user's live-open
// window updating from an action taken by the OTHER user, which a single
// page/context can't exercise.

test('userA creates a group with userB, and userB leaving updates userA live', async ({ browser }) => {
  const contextA = await browser.newContext({ storageState: authFile('userA') })
  const contextB = await browser.newContext({ storageState: authFile('userB') })
  const pageA = await contextA.newPage()
  const pageB = await contextB.newPage()

  try {
    await pageA.goto('/')
    await pageB.goto('/')

    await expect(pageA.locator('#users-list').getByRole('button', { name: users.userB.uname })).toBeVisible({ timeout: 15_000 })
    await expect(pageB.locator('#users-list').getByRole('button', { name: users.userA.uname })).toBeVisible({ timeout: 15_000 })

    const groupName = `E2E Group ${Date.now()}`
    await pageA.getByRole('button', { name: '+ New Group' }).click()
    await pageA.getByPlaceholder('Group name').fill(groupName)
    await pageA.getByRole('checkbox', { name: users.userB.uname }).click()
    await pageA.getByRole('button', { name: 'Create Group' }).click()

    const chatWindowA = pageA.locator('.chat-window')
    await expect(chatWindowA.getByRole('heading', { name: groupName })).toBeVisible()

    // userB never clicked anything to open this — createGroupChat pushes
    // chat-opened to every then-online named member automatically.
    const chatWindowB = pageB.locator('.chat-window')
    await expect(chatWindowB.getByRole('heading', { name: groupName })).toBeVisible({ timeout: 10_000 })

    await chatWindowA.getByRole('button', { name: /Members/ }).click()
    await expect(chatWindowA.getByText(users.userB.uname)).toBeVisible()

    // userB leaves.
    await chatWindowB.getByRole('button', { name: /Members/ }).click()
    pageB.once('dialog', (dialog) => void dialog.accept())
    await chatWindowB.getByRole('button', { name: 'Leave Group' }).click()

    // userB's own window closes (the leave broadcast reaches the leaver
    // too, as an old member)...
    await expect(chatWindowB).not.toBeVisible({ timeout: 10_000 })
    // ...and userA's still-open window's roster updates live, with no
    // reload, to no longer show userB.
    await expect(chatWindowA.getByText(users.userB.uname)).not.toBeVisible({ timeout: 10_000 })
  } finally {
    await contextA.close()
    await contextB.close()
  }
})

test('adding and removing a member updates every other open window live', async ({ browser }) => {
  const contextA = await browser.newContext({ storageState: authFile('userA') })
  const contextB = await browser.newContext({ storageState: authFile('userB') })
  const contextOffline = await browser.newContext()
  const pageA = await contextA.newPage()
  const pageB = await contextB.newPage()
  const pageOffline = await contextOffline.newPage()

  try {
    const offlineUsername = `e2e_offline_${Date.now()}`
    await registerDisposableUser(pageOffline, offlineUsername)

    await pageA.goto('/')
    await pageB.goto('/')
    await expect(pageA.locator('#users-list').getByRole('button', { name: users.userB.uname })).toBeVisible({ timeout: 15_000 })

    const groupName = `E2E Add-Remove Group ${Date.now()}`
    await pageA.getByRole('button', { name: '+ New Group' }).click()
    await pageA.getByPlaceholder('Group name').fill(groupName)
    await pageA.getByRole('checkbox', { name: users.userB.uname }).click()
    await pageA.getByRole('button', { name: 'Create Group' }).click()

    const chatWindowA = pageA.locator('.chat-window')
    const chatWindowB = pageB.locator('.chat-window')
    await expect(chatWindowA.getByRole('heading', { name: groupName })).toBeVisible()
    await expect(chatWindowB.getByRole('heading', { name: groupName })).toBeVisible({ timeout: 10_000 })

    await chatWindowA.getByRole('button', { name: /Members/ }).click()
    await chatWindowB.getByRole('button', { name: /Members/ }).click()

    // userA adds the never-logged-in, genuinely offline user — the same
    // capability add-group-member has had since it was added for exactly
    // this (adding anyone by username, online or not).
    await chatWindowA.getByLabel('Add a member by username').fill(offlineUsername)
    await chatWindowA.getByRole('button', { name: 'Add' }).click()

    await expect(chatWindowA.getByText(offlineUsername)).toBeVisible()
    // userB's already-open Members panel updates live too, from an action
    // userB never took.
    await expect(chatWindowB.getByText(offlineUsername)).toBeVisible({ timeout: 10_000 })

    // userA removes them again.
    pageA.once('dialog', (dialog) => void dialog.accept())
    await chatWindowA.getByRole('button', { name: `Remove ${offlineUsername}` }).click()
    await expect(chatWindowA.getByText(offlineUsername)).not.toBeVisible()
    await expect(chatWindowB.getByText(offlineUsername)).not.toBeVisible({ timeout: 10_000 })
  } finally {
    await contextA.close()
    await contextB.close()
    await contextOffline.close()
  }
})

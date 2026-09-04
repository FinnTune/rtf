import { test, expect } from '@playwright/test'
import { authFile, users } from '../helpers/users'

// Two real browser contexts, not test.use({ storageState }) — this test
// needs both identities live and connected simultaneously, which a single
// page/context can't do.
test('two users exchange messages in real time over a direct chat', async ({ browser }) => {
  const contextA = await browser.newContext({ storageState: authFile('userA') })
  const contextB = await browser.newContext({ storageState: authFile('userB') })
  const pageA = await contextA.newPage()
  const pageB = await contextB.newPage()

  try {
    await pageA.goto('/')
    await pageB.goto('/')

    // Each user's WS connection needs a moment to establish — loading a
    // saved storageState into a fresh context makes checkLogin close the
    // stale connection from auth.setup.ts and mint a fresh OTP before this
    // page's own WS reconnects, a real multi-round-trip handshake, not an
    // instant operation. Wait for each to see the other online before
    // interacting, rather than racing it, with a generous timeout to match.
    await expect(pageA.locator('#users-list').getByRole('button', { name: users.userB.uname })).toBeVisible({
      timeout: 15_000,
    })
    await expect(pageB.locator('#users-list').getByRole('button', { name: users.userA.uname })).toBeVisible({
      timeout: 15_000,
    })

    // A opens a direct chat with B and sends the first message. This also
    // creates the underlying conversation server-side — B doesn't get a
    // window pushed automatically (only a toast + unread marker; opening
    // the direct message table doesn't broadcast) until B opens it too.
    await pageA.locator('#users-list').getByRole('button', { name: users.userB.uname }).click()
    const chatWindowA = pageA.locator('.chat-window')
    const messageFromA = `Hello from A ${Date.now()}`
    await chatWindowA.getByPlaceholder('Type your message').fill(messageFromA)
    await chatWindowA.getByRole('button', { name: 'Send' }).click()
    await expect(chatWindowA.getByText(messageFromA)).toBeVisible()

    // B opens the same conversation from their side — this loads chat
    // history, which should already include A's message.
    await pageB.locator('#users-list').getByRole('button', { name: users.userA.uname }).click()
    const chatWindowB = pageB.locator('.chat-window')
    await expect(chatWindowB.getByText(messageFromA)).toBeVisible()

    // Now that both windows are open, B's reply should arrive in A's
    // already-open window live, with no reload/re-open needed.
    const messageFromB = `Hello from B ${Date.now()}`
    await chatWindowB.getByPlaceholder('Type your message').fill(messageFromB)
    await chatWindowB.getByRole('button', { name: 'Send' }).click()
    await expect(chatWindowA.getByText(messageFromB)).toBeVisible()
  } finally {
    await contextA.close()
    await contextB.close()
  }
})

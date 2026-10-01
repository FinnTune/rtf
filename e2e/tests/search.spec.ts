import { test, expect } from '@playwright/test'
import { authFile, users } from '../helpers/users'

// Both post search (SearchPostsHandler) and message search
// (SearchMessagesHandler) had no E2E coverage at all.

test.describe('post search', () => {
  test.use({ storageState: authFile('userA') })

  test('searching filters the feed to matching titles, and a non-matching query shows the empty state', async ({ page }) => {
    const title = `E2E searchable post ${Date.now()}`
    const content = 'Created by the Playwright E2E suite — safe to ignore.'

    await page.goto('/')
    await page.getByRole('link', { name: 'New Post' }).click()
    await page.locator('#post-title').fill(title)
    await page.locator('#post-content').fill(content)
    await page.locator('#add-post-submit').click()
    await expect(page.getByRole('link', { name: title })).toBeVisible()

    // Scoped to #search-form specifically — MessageSearchPanel's own
    // "Search" button is always present in the sidebar too.
    const searchForm = page.locator('#search-form')

    // Search for a substring of the title, not the whole thing — proves
    // this is a real LIKE search, not an exact-match coincidence.
    const searchSubstring = title.slice(0, title.indexOf(' ', title.indexOf(' ') + 1))
    await searchForm.getByLabel('Search posts').fill(searchSubstring)
    await searchForm.getByRole('button', { name: 'Search' }).click()
    await expect(page.getByRole('link', { name: title })).toBeVisible()

    // A query that can't match anything (this run's own timestamp makes it
    // unique) shows the explicit empty state, not just an absent post.
    const nonMatchingQuery = `nonexistent-${Date.now()}`
    await searchForm.getByLabel('Search posts').fill(nonMatchingQuery)
    await searchForm.getByRole('button', { name: 'Search' }).click()
    await expect(page.getByText(`No posts found for "${nonMatchingQuery}".`)).toBeVisible()
    await expect(page.getByRole('link', { name: title })).not.toBeVisible()

    // Clear returns to the normal, unfiltered feed.
    await searchForm.getByRole('button', { name: 'Clear' }).click()
    await expect(page.getByRole('link', { name: title })).toBeVisible()
  })
})

test.describe('message search', () => {
  // Two real browser contexts, not test.use({ storageState }) — this test
  // needs a real message to exist between two users before searching for
  // it, the same reason chat.spec.ts uses two contexts.
  test('finds a past message by content and opens its conversation on click', async ({ browser }) => {
    const contextA = await browser.newContext({ storageState: authFile('userA') })
    const contextB = await browser.newContext({ storageState: authFile('userB') })
    const pageA = await contextA.newPage()
    const pageB = await contextB.newPage()

    try {
      await pageA.goto('/')
      await pageB.goto('/')

      await expect(pageA.locator('#users-list').getByRole('button', { name: users.userB.uname })).toBeVisible({ timeout: 15_000 })

      const messageText = `E2E searchable message ${Date.now()}`
      await pageA.locator('#users-list').getByRole('button', { name: users.userB.uname }).click()
      const chatWindowA = pageA.locator('.chat-window')
      await chatWindowA.getByPlaceholder('Type your message').fill(messageText)
      await chatWindowA.getByRole('button', { name: 'Send' }).click()
      await expect(chatWindowA.getByText(messageText)).toBeVisible()

      // Close the window before searching — the search result itself (not
      // an already-open window showing the same text) is what's under
      // test here.
      await chatWindowA.getByRole('button', { name: 'Close chat' }).click()

      const searchSubstring = messageText.slice(0, messageText.indexOf(' ', messageText.indexOf(' ') + 1))
      const messageSearchPanel = pageA.locator('#message-search')
      await messageSearchPanel.getByLabel('Search your messages').fill(searchSubstring)
      await messageSearchPanel.getByRole('button', { name: 'Search' }).click()

      const result = pageA.locator('#message-search-results li', { hasText: messageText })
      await expect(result).toBeVisible()

      await result.click()
      await expect(pageA.locator('.chat-window').getByText(messageText)).toBeVisible()
    } finally {
      await contextA.close()
      await contextB.close()
    }
  })
})

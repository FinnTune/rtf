import { test, expect } from '@playwright/test'
import { authFile, registerDisposableUser, users } from '../helpers/users'

// Admin-only flows (category CRUD, user management/ban, user search) had
// no E2E coverage at all — adminUser (helpers/users.ts) is seeded directly
// into the database by setup/run-server.sh specifically to make this
// possible, since there was previously no way to reach an admin account
// through normal E2E browser interaction (promoting a user to admin only
// happens once, at server startup).
//
// Neither test uses test.use({ storageState }) + the page/context fixture —
// every context here is created manually with browser.newContext() and
// closed explicitly in a finally block instead. This isn't just style:
// a test using the page/context fixture with a saved storageState, once its
// fixture teardown runs at test end, was found to leave the NEXT test's
// freshly-created context stuck forever on its very first render (blank
// page, a click that never becomes actionable) — confirmed by bisection to
// require exactly that combination (fixture-based context + automatic
// teardown immediately before a new browser.newContext() elsewhere), and
// confirmed absent when every context in the file is instead opened and
// closed by hand, which is what both tests below do. Likely a Playwright/
// headless-shell trace-or-screenshot-capture quirk around fixture teardown
// timing, not a bug in the app itself.

test('an admin can create, rename, and delete a category', async ({ browser }) => {
  const adminContext = await browser.newContext({ storageState: authFile('adminUser'), ignoreHTTPSErrors: true })
  const page = await adminContext.newPage()

  try {
    await page.goto('/')
    await page.getByRole('link', { name: 'Manage Categories' }).click()
    await expect(page.getByRole('heading', { name: 'Manage Categories' })).toBeVisible()

    // Scoped to #manage-categories throughout — the sidebar's CategoryNav
    // also renders every category name as its own nav button, so an
    // unscoped getByText would be ambiguous.
    const managePage = page.locator('#manage-categories')

    const categoryName = `E2E Category ${Date.now()}`
    await managePage.getByLabel('New category name').fill(categoryName)
    await managePage.getByRole('button', { name: 'Add Category' }).click()
    await expect(managePage.getByText(categoryName)).toBeVisible()

    const categoryRow = managePage.locator('.manage-category-row', { hasText: categoryName })
    await categoryRow.getByRole('button', { name: 'Rename' }).click()
    // Short enough to fit category_name's 30-character limit alongside the
    // timestamp suffix — category.category_name VARCHAR(30) (see
    // database/createTables.sql), also enforced client-side via this
    // input's own maxLength.
    const renamedCategory = `E2E Cat R ${Date.now()}`
    // Not scoped to categoryRow beyond this point — like a comment's edit
    // mode (see comments-and-reactions.spec.ts), renaming swaps the row's
    // name to an <input>, whose value isn't part of element text content,
    // so categoryRow's hasText filter stops matching once inside it. Only
    // one row is ever in edit mode at a time, so #manage-categories's own
    // "Category name"/"Save" locators are unambiguous on their own. exact:
    // true since "Category name" would otherwise substring-match the
    // create form's "New category name" label too.
    await managePage.getByLabel('Category name', { exact: true }).fill(renamedCategory)
    await managePage.getByRole('button', { name: 'Save' }).click()
    await expect(managePage.getByText(renamedCategory)).toBeVisible()
    await expect(managePage.getByText(categoryName, { exact: true })).not.toBeVisible()

    page.once('dialog', (dialog) => void dialog.accept())
    await managePage.locator('.manage-category-row', { hasText: renamedCategory }).getByRole('button', { name: 'Delete' }).click()
    await expect(managePage.getByText(renamedCategory)).not.toBeVisible()
  } finally {
    await adminContext.close()
  }
})

test('an admin can search for a user and ban/unban them', async ({ browser }) => {
  // Longer than the default 30s — this test deliberately waits out the
  // shared auth rate limiter's refill (see below) before registering its
  // disposable ban target.
  test.setTimeout(45_000)

  // auth.setup.ts's three identities (userA/userB registered+logged-in,
  // adminUser logged in) already spend all 5 tokens of the shared per-IP
  // auth rate limiter's burst, and this test runs immediately after setup
  // (and the category-CRUD test above, which spends no auth tokens of its
  // own) — with no other slower test in between to let a token refill
  // naturally. Registering one more (disposable) user below needs a 6th
  // token; the limiter refills at 1 token per 12 seconds (see
  // utility/ratelimit.go's authLimiter), so wait that out first.
  await new Promise((resolve) => setTimeout(resolve, 13_000))

  // A disposable target, not userB — banning provably isn't undone by a
  // later unban (see registerDisposableUser's doc comment: kickUser
  // removes the in-memory session entry checkLogin needs to accept a
  // saved storageState at all), so banning one of the three shared
  // fixture identities here would silently break every other spec file
  // that reuses its storageState for the rest of this run.
  const adminContext = await browser.newContext({ storageState: authFile('adminUser'), ignoreHTTPSErrors: true })
  const registerContext = await browser.newContext({ ignoreHTTPSErrors: true })
  const page = await adminContext.newPage()
  const registerPage = await registerContext.newPage()

  try {
    const disposableUsername = `e2e_ban_target_${Date.now()}`
    await registerDisposableUser(registerPage, disposableUsername)

    await page.goto('/')
    await page.getByRole('link', { name: 'Manage Users' }).click()
    await expect(page.getByRole('heading', { name: 'Manage Users' })).toBeVisible()

    // Scoped to #manage-users throughout — the page also always renders
    // the post-search form and MessageSearchPanel, both with their own
    // same-named "Search" button.
    const managePage = page.locator('#manage-users')

    // Search narrows the list to just the matching user (#154) — proves
    // this is a real filtered search, not just "the user happens to be on
    // the first page".
    await managePage.getByLabel('Search users by username or email').fill(disposableUsername)
    await managePage.getByRole('button', { name: 'Search' }).click()
    await expect(managePage.getByText(`${disposableUsername}@example.com`)).toBeVisible()
    await expect(managePage.getByText(users.adminUser.email)).not.toBeVisible()

    const targetRow = managePage.locator('.manage-category-row', { hasText: disposableUsername })
    page.once('dialog', (dialog) => void dialog.accept())
    await targetRow.getByRole('button', { name: 'Ban' }).click()
    await expect(targetRow.getByText('— banned')).toBeVisible()
    await expect(targetRow.getByRole('button', { name: 'Unban' })).toBeVisible()

    await targetRow.getByRole('button', { name: 'Unban' }).click()
    await expect(targetRow.getByText('— banned')).not.toBeVisible()
  } finally {
    await adminContext.close()
    await registerContext.close()
  }
})

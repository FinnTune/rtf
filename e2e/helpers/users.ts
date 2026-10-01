import type { Page } from '@playwright/test'
import { expect } from '@playwright/test'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// Fixed identities, not randomly generated: the E2E database is wiped and
// recreated fresh on every run (see setup/run-server.sh), so there's no
// cross-run collision risk, and fixed names make failures easier to read.
// userA/userB are registered+logged-in exactly once each; adminUser is
// seeded directly and only logged in (see tests/auth.setup.ts) — the rest
// of the suite reuses all three via storageState instead of authenticating
// again (see playwright.config.ts's comment on the 'setup' project for
// why: the shared per-IP auth rate limiter has a burst of only 5, and
// these three identities' setup already spends 2+2+1 of it).
// A test that itself needs one more real, registered-but-never-logged-in
// identity (see group-chat-membership.spec.ts's offline-member add/remove
// coverage) spends one more token of that same shared budget — keep that
// in mind before adding another one.
export const users = {
  userA: {
    fname: 'E2E',
    lname: 'UserA',
    uname: 'e2e_user_a',
    email: 'e2e_user_a@example.com',
    age: '30',
    gender: 'other',
    password: 'E2ePassword123',
  },
  userB: {
    fname: 'E2E',
    lname: 'UserB',
    uname: 'e2e_user_b',
    email: 'e2e_user_b@example.com',
    age: '30',
    gender: 'other',
    password: 'E2ePassword123',
  },
  // Not registered through the UI like userA/userB — seeded directly into
  // the database by setup/run-server.sh (and promoted to the admin role by
  // migrate()'s own "a user literally named admin" convenience), so
  // auth.setup.ts only needs to log this one in, not register it too.
  // These values must match the INSERT in run-server.sh exactly.
  adminUser: {
    fname: 'E2E',
    lname: 'Admin',
    uname: 'admin',
    email: 'e2e_admin@example.com',
    age: '30',
    gender: 'other',
    password: 'E2eAdminPassword123',
  },
} as const

export const authFile = (name: keyof typeof users) => path.join(__dirname, '..', '.auth', `${name}.json`)

// Registers (but deliberately does not log in — callers never need this
// identity online) a brand-new, disposable user directly through the real
// register form, so CSRFProtect's Origin check is satisfied the same way
// a real browser request satisfies it (a bare API POST from here, with no
// Origin header, would be rejected outright). Consumes exactly one token
// from the shared per-IP auth rate limiter — see this file's own doc
// comment on why that budget is tight.
//
// Also the right tool for anything a test needs to mutate destructively
// (e.g. banning): mutating one of the three shared fixture identities
// above instead would corrupt it for every other spec file reusing its
// storageState for the rest of the run — banning in particular is
// provably NOT undone by a later unban, since SetUserBannedHandler's
// kickUser call removes the in-memory session entry checkLogin relies on
// to accept that identity's saved storageState at all, and unbanning only
// flips the database's banned column back, never restoring that entry.
export async function registerDisposableUser(page: Page, uname: string) {
  await page.goto('/')
  await page.getByRole('button', { name: 'Register' }).click()
  const registerForm = page.locator('form.register-form')
  await registerForm.locator('#regfname').fill('E2E')
  await registerForm.locator('#reglname').fill('Disposable')
  await registerForm.locator('#reguname').fill(uname)
  await registerForm.locator('#regemail').fill(`${uname}@example.com`)
  await registerForm.locator('#regage').fill('30')
  await registerForm.locator('#reggender').selectOption('other')
  await registerForm.locator('#regpassword').fill('E2ePassword123')
  await registerForm.locator('#regconfpassword').fill('E2ePassword123')
  await registerForm.getByRole('button', { name: 'Register' }).click()
  // RegisterForm switches straight to the login view on success — confirms
  // registration actually succeeded, without this test ever logging in.
  await expect(page.locator('form.login-form')).toBeVisible()
}

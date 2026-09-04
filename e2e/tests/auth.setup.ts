import { test as setup, expect, type Page } from '@playwright/test'
import { authFile, users } from '../helpers/users'

async function registerAndLogIn(page: Page, user: (typeof users)[keyof typeof users]) {
  await page.goto('/')

  // LoggedOutShell's header (Register/Login toggle buttons) is always
  // rendered alongside whichever form is showing, so "Register"/"Login" is
  // never a unique accessible name on its own once a form is up — every
  // form-submit lookup below is scoped to its form to disambiguate from
  // the header's same-named toggle button.
  await page.getByRole('button', { name: 'Register' }).click()
  const registerForm = page.locator('form.register-form')
  await registerForm.locator('#regfname').fill(user.fname)
  await registerForm.locator('#reglname').fill(user.lname)
  await registerForm.locator('#reguname').fill(user.uname)
  await registerForm.locator('#regemail').fill(user.email)
  await registerForm.locator('#regage').fill(user.age)
  await registerForm.locator('#reggender').selectOption(user.gender)
  await registerForm.locator('#regpassword').fill(user.password)
  await registerForm.locator('#regconfpassword').fill(user.password)
  await registerForm.getByRole('button', { name: 'Register' }).click()

  // RegisterForm's onRegistered callback switches straight to the login
  // view — registration alone never logs the user in.
  const loginForm = page.locator('form.login-form')
  await expect(loginForm).toBeVisible()
  await loginForm.locator('#username').fill(user.uname)
  await loginForm.locator('#password').fill(user.password)
  await loginForm.getByRole('button', { name: 'Login' }).click()

  // Confirms the whole register -> login -> authenticated-session round
  // trip actually worked, not just that no error was thrown along the way.
  await expect(page.locator('#topbar-username')).toHaveText(user.uname)
}

setup('register and log in userA', async ({ page }) => {
  await registerAndLogIn(page, users.userA)
  await page.context().storageState({ path: authFile('userA') })
})

setup('register and log in userB', async ({ page }) => {
  await registerAndLogIn(page, users.userB)
  await page.context().storageState({ path: authFile('userB') })
})

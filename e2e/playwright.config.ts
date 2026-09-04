import { defineConfig, devices } from '@playwright/test'

const baseURL = 'https://localhost:8543'

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  // A single shared server + database backs the whole suite (no per-test
  // isolation), and re-authenticating a saved storageState in a fresh
  // context triggers a real close-old-connection/mint-OTP/WS-reconnect
  // round trip that takes a few real seconds — running specs concurrently
  // just makes every one of those slower by contending for the same CPU.
  // A smoke suite this small doesn't need the speed badly enough to trade
  // away that reliability.
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['html', { open: 'never' }]] : 'list',
  timeout: 30_000,

  use: {
    baseURL,
    // The app's TLS cert is a locally-generated self-signed one (see
    // setup/run-server.sh) — without this, Chromium hard-blocks navigation
    // on the very first page.goto() with a cert-interstitial.
    ignoreHTTPSErrors: true,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      // A real test, not the older globalSetup.ts function API — its own
      // pass/fail shows up in the report. Registers/logs in the suite's
      // fixed identities once and saves storageState for the other
      // projects to reuse, so the rest of the suite never calls
      // /register or /login again (see tests/auth.setup.ts for why: the
      // shared per-IP auth rate limiter has a burst of only 5).
      name: 'setup',
      testMatch: /.*\.setup\.ts/,
      use: { ...devices['Desktop Chrome'], ignoreHTTPSErrors: true },
    },
    {
      name: 'chromium',
      testMatch: /.*\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
      dependencies: ['setup'],
    },
  ],

  webServer: {
    command: './setup/run-server.sh',
    url: `${baseURL}/healthz`,
    // Separate knob from use.ignoreHTTPSErrors above — this one governs
    // Playwright's own readiness-check request, not browser navigation.
    ignoreHTTPSErrors: true,
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
})

import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// Fixed identities, not randomly generated: the E2E database is wiped and
// recreated fresh on every run (see setup/run-server.sh), so there's no
// cross-run collision risk, and fixed names make failures easier to read.
// Exactly two identities, registered+logged-in exactly once each (see
// tests/auth.setup.ts) — the rest of the suite reuses these via
// storageState instead of logging in again (see playwright.config.ts's
// comment on the 'setup' project for why: the shared per-IP auth rate
// limiter has a burst of only 5).
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
} as const

export const authFile = (name: keyof typeof users) => path.join(__dirname, '..', '.auth', `${name}.json`)

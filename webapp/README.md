# webapp

React + TypeScript + Vite frontend for theDialectic. `npm run build` produces
`webapp/dist`, which the Go server (`main.go`, `distDir = "./webapp/dist"`)
serves directly as static assets — there's no separate frontend server in
production.

See the repository root [README](../README.md) for how this fits into the
app as a whole (routes, WebSocket events, running the full stack, Docker).

## Scripts

- `npm run dev` — Vite dev server with HMR; proxies API/WebSocket calls to
  the real backend at `https://localhost:8443` (run `go run .` or
  `docker compose up` separately from the repo root). The proxy's route
  list (`vite.config.ts`) is a manually-kept allowlist, not derived from
  `main.go` — a new backend route needs a matching entry added there too,
  or `npm run dev` will silently fail to reach it (Vite serves its own
  404/index.html instead of proxying through). `npm run build` against a
  real running server skips this allowlist entirely and is the more
  reliable way to check a change actually works end-to-end — the root
  README's E2E suite does exactly that.
- `npm run build` — typecheck + production build to `dist/`.
- `npm run lint` / `npm run typecheck` / `npm run test`

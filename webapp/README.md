# webapp

React + TypeScript + Vite frontend for theDialectic, replacing the vanilla-JS
`frontend/` app. See `/home/quetzalcoatl/.claude/plans/kind-whistling-taco.md`
(or the project's PR history) for the migration plan and phasing.

Not yet wired into the Go server (`main.go` still serves the old `frontend/`
app) — this project builds and tests independently until the cutover PR.

## Scripts

- `npm run dev` — Vite dev server with HMR; proxies API/WebSocket calls to
  the real backend at `https://localhost:8443` (run `go run .` or
  `docker compose up` separately).
- `npm run build` — typecheck + production build to `dist/`.
- `npm run lint` / `npm run typecheck` / `npm run test`

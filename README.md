# Real-Time Forum

Created by André J. Teetor as a learning project to explore:

- HTTP + session-based auth
- WebSocket real-time messaging
- Single-page app style frontend (vanilla JavaScript)
- SQLite persistence
- basic security and state management patterns

This repository runs a forum with posts/comments plus private chat and online-user presence.

![Screenshot](picture_test.png)

## Tech Stack

- Backend: Go (`net/http`, `gorilla/websocket`)
- Database: SQLite (`github.com/mattn/go-sqlite3`)
- Frontend: HTML/CSS + vanilla JS modules
- Transport: HTTPS + WSS

## Repository Layout

- `main.go`: application entrypoint, HTTP routes, TLS server startup
- `websocket/`: HTTP handlers, WS manager/client/event logic
- `database/`: DB open/init helpers and SQL schema files
- `frontend/`: static SPA assets (`index.html`, CSS, JS)
- `utility/`: password hashing, cookie helpers
- `tests/`: automated test suites and shared test helpers
- `logfiles/`: runtime logs

## Prerequisites

- Go (1.20+ recommended)
- Node.js + npm (for linting frontend code)
- OpenSSL (if generating local TLS certificates)

## Quick Start

From repository root:

```bash
PORT=8443 go run .
```

Then open:

- [https://localhost:8443](https://localhost:8443)

## Runtime Configuration

Environment variables:

- `PORT` (default: `8443`)
- `TLS_CERT` (default: `localhost.crt`)
- `TLS_KEY` (default: `localhost.key`)
- `ALLOWED_ORIGIN` (default: `https://localhost:8443`)

Example with explicit cert paths:

```bash
PORT=8443 TLS_CERT=localhost.crt TLS_KEY=localhost.key ALLOWED_ORIGIN=https://localhost:8443 go run .
```

## TLS Certificates (Local Development)

If certificate files do not exist, generate local ones:

```bash
openssl req -new -newkey rsa:2048 -nodes -keyout localhost.key -out localhost.csr
openssl x509 -req -days 365 -in localhost.csr -signkey localhost.key -out localhost.crt
```

Your browser may warn for self-signed certs; accept locally for development.

## Database

The app uses SQLite at `database/forum.db`.

- On startup, the app opens the DB connection.
- Schema/seed SQL lives in `database/createTables.sql`.
- The seeded categories and example posts support immediate local testing.

## Core Features

- User registration and login
- Session cookie auth (`session_id`)
- Post creation and category filtering
- Post comments
- Real-time private chat via WebSocket
- Live online-user list
- Typing indicator events
- Chat history pagination

## HTTP / WS Endpoints

HTTP:

- `GET /` - serves SPA entrypoint
- `POST /register`
- `POST /login`
- `POST /logout`
- `GET /checkLogin`
- `GET /getAllPosts`
- `POST /addPost`
- `POST /getPostsByCategory`
- `POST /addcomment`
- `GET /comments?postId=<id>`

WebSocket:

- `GET /ws?otp=<token>`

Event types include:

- `user-connect`
- `new-message` / `sent-message`
- `users-online`
- `typing` / `stop-typing`
- `get-chat-history` / `get-more-chat-history`
- `chat_history`

## Security Notes

Recent hardening includes:

- write endpoints (`/addPost`, `/addcomment`) now derive identity from authenticated session state rather than trusting client-sent user identity fields
- multiple frontend user-content render paths were moved from unsafe `innerHTML` usage to safer text-based rendering
- request-path fatal exits were removed from handlers in favor of safe HTTP error responses
- origin checks are now configurable through `ALLOWED_ORIGIN`

Recommended next security improvements:

- CSRF protection for state-changing routes
- tighter input length/validation constraints at API boundaries
- cookie/session expiration and refresh policy review
- rate limiting for login and write endpoints

## Development Workflow

Run server:

```bash
PORT=8443 go run .
```

Run all backend tests (requires CGO for SQLite):

```bash
CGO_ENABLED=1 CGO_CFLAGS="-Wno-discarded-qualifiers" go test ./...
```

Run frontend lint:

```bash
npm run lint
```

Run the full local check (same as CI):

```bash
npm run lint
CGO_ENABLED=1 CGO_CFLAGS="-Wno-discarded-qualifiers" go test ./...
```

## Continuous Integration

GitHub Actions runs on every push and pull request to `master` (see `.github/workflows/test.yml`):

- `npm ci` + `npm run lint`
- `go test ./...` with CGO enabled for SQLite

Dependabot (`.github/dependabot.yml`) opens weekly PRs for Go module, npm, and GitHub Actions updates.

## Testing

The backend has **37 automated tests** under `tests/`. Tests use an in-memory SQLite database so they do not touch `database/forum.db`.

### Test layout

```
tests/
├── testutil/          # shared in-memory DB setup
├── utility/           # password and cookie tests
└── websocket/         # HTTP handler, OTP, and WS event tests
```

| Location | Files | What is covered |
|----------|-------|-----------------|
| `tests/testutil/` | `database.go` | Shared in-memory SQLite schema and seed data |
| `tests/utility/` | `utility_test.go` | Password hashing/verification, session cookie creation and detection |
| `tests/websocket/` | `handlers_test.go` | Origin checks, post/comment auth enforcement |
| `tests/websocket/` | `handlers_auth_test.go` | Registration, login, logout, session status |
| `tests/websocket/` | `handlers_posts_test.go` | Posts, comments, category filtering |
| `tests/websocket/` | `otp_test.go` | One-time password creation, verification, expiry |
| `tests/websocket/` | `ws_events_test.go` | Chat messages, user presence, typing indicators, chat history |

Internal test hooks used by `tests/websocket/` live in `websocket/testhooks.go`.

### Coverage areas

- **Security**: WebSocket origin validation, session-based identity for writes (client-sent user IDs are ignored)
- **Auth**: Registration, login (valid/invalid credentials), logout, login status checks
- **Forum API**: Listing posts, filtering by category, adding posts with categories, reading and adding comments
- **Real-time**: OTP lifecycle, chat message persistence, online user broadcasts, typing/stop-typing events, chat history pagination

### Commands

Run all tests with coverage:

```bash
CGO_ENABLED=1 CGO_CFLAGS="-Wno-discarded-qualifiers" go test ./... -cover
```

Run a single test package:

```bash
CGO_ENABLED=1 go test ./tests/utility -v
CGO_ENABLED=1 go test ./tests/websocket -v
```

Run one test by name:

```bash
CGO_ENABLED=1 go test ./tests/websocket -run TestLoginHandler_Success -v
```

Current coverage (approximate, measured against source packages):

- `utility/`: ~93%
- `websocket/`: ~63%

Not yet covered: full WebSocket upgrade handshake, live connection read/write loops, and the `main`/`database`/`logfiles` packages.

## Troubleshooting

### 1) `go-sqlite3` warning during build

On some Linux toolchains you may see:

`assignment discards 'const' qualifier from pointer target type`

This comes from SQLite C bindings and does not usually block startup.

Optional local suppression:

```bash
CGO_CFLAGS="-Wno-discarded-qualifiers" PORT=8443 go run .
```

### 2) TLS handshake / browser certificate errors

- confirm cert/key paths match `TLS_CERT` and `TLS_KEY`
- ensure browser is using `https://localhost:<PORT>`
- accept self-signed cert for local dev

### 3) WebSocket fails to connect

- verify app is running over HTTPS (WSS requires secure context)
- confirm `ALLOWED_ORIGIN` matches browser origin exactly
- verify websocket URL host/port matches server

### 4) Port already in use

Use another local port:

```bash
PORT=9443 go run .
```

## Known Limitations

- no production-grade deployment configuration yet
- frontend uses direct DOM manipulation and can benefit from modular refactoring
- WebSocket connection lifecycle and server startup code are not yet covered by tests

## Contributing

1. Create a feature branch.
2. Keep changes small and focused.
3. Run lint/tests before opening a PR:
   - `npm run lint`
   - `CGO_ENABLED=1 CGO_CFLAGS="-Wno-discarded-qualifiers" go test ./...`
4. Include reproduction steps or test notes for bug fixes.

## License

Copyright (C) 2026 Andre Teetor

This project is licensed under the GNU General Public License v2.0 —
see the [LICENSE](LICENSE) file for details.
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VibePlatform is a platform that lets users vibe-code and deploy applications. Users write code with AI assistance, then the platform handles testing, pushing to GitHub, and deployment.

## Tech Stack

- **Backend**: Go
- **Frontend**: Vite + Vue.js

## Implementation Scope

Features are tracked in `todo.md`. Marker legend:
- `[x]` = must implement
- `[ ]` = skip for now
- `[done]` = already implemented, ignore

Current scope (from todo.md):
- GitHub OAuth registration/login
- AI-assisted test validation
- Automated GitHub push

Deployment and other features are explicitly out of scope for now.

## Docker Image

The code-server containers use a custom image with Node 24, Go 1.24, and Claude Code pre-installed. Build it once before running the server:

```bash
docker build -t vibeplatform-code-server:latest ./docker
```

## Development Commands

**Backend (Go)**
```bash
go run ./cmd/server              # run the server
go test ./...                    # run all tests
go test ./internal/auth/...      # run tests in a specific package
go build ./...                   # build
```

**Frontend (Vite/Vue.js)**
```bash
cd frontend && npm install && npm run dev   # install and start dev server
cd frontend && npm run build               # production build
cd frontend && npx playwright test         # run e2e tests
```

**Environment**: copy `.env.example` to `.env`.

## Environment Variables

```
GITHUB_CLIENT_ID      # GitHub OAuth App ID
GITHUB_CLIENT_SECRET  # GitHub OAuth App Secret
JWT_SECRET            # >=32 chars
PORT                  # default 3001
FRONTEND_URL          # CORS allow-list origin (e.g. http://localhost:5173)
BASE_URL              # Used to construct OAuth callback URL
DATA_DIR              # JSON file storage path (default ./data)
```

Frontend (`frontend/.env`): `VITE_BACKEND_URL` — backend base URL (optional, defaults to same origin).

## Architecture

### Request Path

`cmd/server/main.go` → `internal/server/server.go` (Chi router, Logger/Recoverer/CORS middleware) → `internal/handler/*.go`

### Authentication

1. `/auth/github` redirects user to GitHub OAuth with CSRF state cookie (5-min TTL).
2. `/auth/github/callback` exchanges code for token, fetches GitHub user info, upserts user into store, issues JWT.
3. JWT stored as `session` cookie (HttpOnly, 7-day TTL).
4. All protected handlers call `resolveGithubID()` in `internal/handler/auth_helper.go` to validate the session and extract the GitHub user ID.

Key files: `internal/auth/github.go`, `internal/auth/session.go`, `internal/handler/auth.go`, `internal/handler/auth_helper.go`.

### Storage Layer

All stores live in `internal/store/`. Pattern: mutex-protected in-memory map, loaded from a JSON file at init (missing file = empty, no error), `flush()` writes the full map back on every write. `DATA_DIR` env sets the storage directory.

- `user.go` — in-memory only (ephemeral across restarts); maps by user ID and GitHub ID.
- `container.go` — file-backed; maps userID → ContainerInfo (legacy ubuntu containers).
- `project.go` — file-backed; maps userID → projectName → ProjectInfo.
- `settings.go` — file-backed; maps userID → UserSettings (Anthropic API key).

### Docker Integration

`internal/docker/client.go` wraps the Docker API.

- **Code-server containers**: image `codercom/code-server:latest`, named `vibe-{userID}-{projectName}`, port 8080 bound to an ephemeral host port, `ANTHROPIC_API_KEY` injected via env, workspace at `/home/coder/project`.
- **Legacy ubuntu containers**: image `ubuntu:22.04`, SSH on a random host port.

### Proxy Handler

`internal/handler/proxy.go` reverse-proxies requests to running code-server containers. Detects WebSocket upgrades (`Upgrade: websocket` header) and handles them via a raw TCP tunnel; plain HTTP goes through `httputil.ReverseProxy`. Strips the `/project/{name}` prefix before forwarding.

## Handler & Store Patterns

When adding a new feature, follow the existing conventions:

**Handler**: dependency-injection struct with a constructor (`NewXyzHandler(...) *XyzHandler`), auth via `resolveGithubID()`, JSON request/response with `json.NewDecoder`/`json.NewEncoder`, consistent HTTP status codes.

**Store**: mutex + in-memory map + `flush()` to JSON on every write; load from file at init.

**Project name validation**: regex `^[a-z0-9][a-z0-9-]{0,30}$` (enforced in project handler).

## Frontend Architecture

Single-file SPA: `frontend/src/App.vue` (Vue 3 Composition API). Three states: `loading`, `loggedOut`, `loggedIn`. All API calls use `fetch` with `credentials: 'include'` for cookie-based auth. The backend URL is read from `VITE_BACKEND_URL` at build time.

## Rules

From `rules.md` — must be followed on every feature:
1. Write unit tests before implementing.
2. Run `go test ./...` after implementing and confirm all pass.

## API Reference

Endpoints are documented in `api.md`.

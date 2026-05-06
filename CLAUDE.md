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

## Development Commands

**Backend (Go)**
```bash
go run ./cmd/server       # run the server
go test ./...             # run all tests
go test ./internal/auth/... # run tests in a specific package
go build ./...            # build
```

**Frontend (Vite/Vue.js)**
```bash
cd frontend && npm install && npm run dev   # install and start dev server
cd frontend && npm run build               # production build
```

**Environment**: copy `.env.example` to `.env` and fill in `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `JWT_SECRET`.

## Rules

From `rules.md` — must be followed on every feature:
1. Write unit tests before implementing
2. Run `go test ./...` after implementing and confirm all pass

## api record in api.md
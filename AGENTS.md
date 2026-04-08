# AGENTS.md — Soft Serve

> AI agent guide for [Soft Serve](https://github.com/charmbracelet/soft-serve), a self-hosted Git server with SSH, HTTP, and native git protocol support, plus a built-in TUI.

## Project Identity

- **Module:** `github.com/charmbracelet/soft-serve`
- **Language:** Go 1.25+
- **Binary:** `soft` (entry point: `cmd/soft/main.go`)
- **Database:** SQLite (default) or PostgreSQL, via `jmoiron/sqlx`
- **CLI framework:** `spf13/cobra` (used for both the main CLI and SSH subcommands)
- **SSH framework:** `charm.land/wish/v2`
- **TUI framework:** `charm.land/bubbletea/v2` + `lipgloss/v2` + `glamour/v2`
- **HTTP router:** `gorilla/mux`
- **Config:** YAML file (`$SOFT_SERVE_DATA_PATH/config.yaml`) + env vars (`SOFT_SERVE_*` prefix)

## Architecture Overview

Soft Serve runs **four concurrent servers** via `errgroup`:

| Server | Package | Protocol | Purpose |
|--------|---------|----------|---------|
| SSH | `pkg/ssh/` | SSH | Git push/pull, interactive TUI, admin CLI commands |
| HTTP | `pkg/web/` | HTTP/HTTPS | Git smart HTTP, LFS API, go-get meta, health check |
| Git Daemon | `pkg/daemon/` | Native git (TCP) | Read-only anonymous git access |
| Stats | `pkg/stats/` | HTTP | Prometheus `/metrics` endpoint |

### Layered Design

```
cmd/soft/         → CLI entry points (serve, browse, admin, hook)
pkg/ssh/          → SSH server + middleware + command dispatch
pkg/web/          → HTTP server + middleware + routes
pkg/daemon/       → Git daemon (raw TCP, pktline)
pkg/backend/      → Central business logic orchestrator
pkg/store/        → Data store interface (7 sub-interfaces)
pkg/store/database/ → SQL implementation of Store
pkg/db/           → Database layer (open, migrate, models)
pkg/proto/        → Core domain interfaces (User, Repository)
pkg/access/       → Access level enum (NoAccess → Admin)
pkg/git/          → Git service handlers (upload-pack, receive-pack)
pkg/config/       → Configuration parsing (YAML + env)
pkg/ui/           → TUI components (Bubble Tea models)
```

### Canonical Data Flow

```
Transport layer (SSH/HTTP/daemon)
  → Middleware (context injection, auth, logging)
    → Command/Route dispatch
      → Backend (business logic)
        → Store (data access)
          → Database (SQLite/PostgreSQL)
```

## Directory Structure

```
cmd/
  soft/main.go              — Binary entry point
  soft/serve/               — `soft serve` command (starts all servers)
  soft/admin/               — `soft admin` SSH admin commands
  soft/browse/              — `soft browse` TUI browser
  soft/hook/                — `soft hook` git hook handler
  cmd.go                    — Shared InitBackendContext / CloseDBContext helpers
git/                        — Low-level git operations (repo, commit, tree, tag, refs)
pkg/
  access/                   — AccessLevel enum (NoAccess, ReadOnly, ReadWrite, Admin)
  backend/                  — Central Backend struct: users, repos, auth, LFS, webhooks, cache
  config/                   — Config struct (YAML + env, SOFT_SERVE_* prefix)
  cron/                     — Cron scheduler wrapper (robfig/cron)
  daemon/                   — Git daemon server (TCP, native git protocol)
  db/                       — Database abstraction (open, migrations, models)
    migrate/                — SQL migration files (SQLite + PostgreSQL variants)
    models/                 — DB model structs
  git/                      — Git service handlers (upload-pack, receive-pack, LFS transfer)
  hooks/                    — Git hook generation and interface
  jobs/                     — Cron job definitions (mirror pull)
  jwk/                      — JSON Web Key support
  lfs/                      — Git LFS protocol (client, pointers, scanner, transfers)
  log/                      — Logger setup
  proto/                    — Core interfaces (Repository, User, AccessToken, errors)
  ssh/                      — SSH server, middleware, session handler
    cmd/                    — 27 SSH subcommands (git, repo, user, token, webhook, settings)
  sshutils/                 — SSH utility functions
  ssrf/                     — SSRF protection for outbound requests
  stats/                    — Prometheus metrics server
  storage/                  — Storage abstraction (file storage for LFS)
  store/                    — Store interface (composite of 7 sub-interfaces)
    database/               — SQL-backed Store implementation
  sync/                     — Sync utilities
  task/                     — Async task manager
  ui/                       — Bubble Tea TUI components
    common/                 — Shared TUI state/styles
    components/             — Reusable widgets (code, footer, header, selector, statusbar, tabs, viewport)
    pages/                  — TUI pages (repo list, repo detail)
    styles/                 — TUI styling
  utils/                    — General utilities (SanitizeRepo, ValidateRepo)
  version/                  — Version info
  web/                      — HTTP server (git smart HTTP, LFS API, auth, health)
  webhook/                  — Webhook event system (push, branch/tag, repo, collaborator)
testscript/                 — Integration tests (testscript framework)
migrations/                 — Goose SQL migrations
```

## Key Patterns

### Context-Based Dependency Injection

All major dependencies are threaded through `context.Context` with typed keys:

```go
// Injecting
ctx = config.WithContext(ctx, cfg)
ctx = backend.WithContext(ctx, be)

// Extracting
cfg := config.FromContext(ctx)
be := backend.FromContext(ctx)
```

Shared bootstrap in `cmd/cmd.go:InitBackendContext` (used as Cobra `PersistentPreRunE`).

### Backend as Central Orchestrator

`pkg/backend/backend.go:Backend` is the single entry point for all business logic. It wraps the Store and provides methods for repos, users, auth, LFS, webhooks, settings, and caching. Handlers and commands call Backend methods — never the Store directly.

### Store Interface (Composite)

`pkg/store/store.go:Store` aggregates 7 sub-interfaces:

- `RepositoryStore` — repo CRUD
- `UserStore` — user CRUD
- `CollaboratorStore` — repo collaborators
- `SettingStore` — server settings
- `LFSStore` — LFS objects/locks
- `AccessTokenStore` — API tokens
- `WebhookStore` — webhook management

Single SQL implementation in `pkg/store/database/`.

### Authentication

**SSH:** Public key verification → `be.UserByPublicKey()`. Keyboard-interactive as keyless fallback.

**HTTP:** Three schemes via `Authorization` header:
- `Basic` — username/password (bcrypt) or username/access-token
- `Token` — access token directly (`ss_` prefixed, SHA256 hashed)
- `Bearer` — JWT (Ed25519 signed, scoped to repo)

### Authorization

Four levels in `pkg/access/access.go`:
- `NoAccess` (0) — denied
- `ReadOnlyAccess` (1) — clone/fetch
- `ReadWriteAccess` (2) — push, LFS locks
- `AdminAccess` (3) — server management

### SSH Command Dispatch

Non-PTY SSH sessions go through `CommandMiddleware` which builds a Cobra command tree from `pkg/ssh/cmd/`. The same `cobra.Command` pattern used for the main CLI is reused for SSH commands.

PTY sessions launch the Bubble Tea TUI via `SessionHandler`.

### Git Operations

Git push/pull is handled by shelling out to the `git` binary (not pure Go) via `pkg/git/service.go:gitServiceHandler`. Environment variables pass context (repo name, username, public key) to git hooks.

### Webhook System

Events defined in `pkg/webhook/`: push, branch create/delete, tag create/delete, repository create/delete, collaborator add/remove. Delivery via HTTP POST with HMAC-SHA256 signatures. SSRF protection in `pkg/ssrf/`.

## Build & Test

```bash
# Build
go build ./cmd/soft

# Run all tests
go test ./...

# Run integration tests
go test ./testscript/...

# Run specific test
go test -run TestName ./pkg/backend/...
```

### Test Framework

- Unit tests: standard Go `testing` + `matryer/is` for assertions
- Integration tests: `rogpeppe/go-internal/testscript` (script-driven, in `testscript/`)
- Test helpers in `pkg/test/`

## Database

- **Drivers:** SQLite (`modernc.org/sqlite`, pure Go) and PostgreSQL (`lib/pq`)
- **Access:** `jmoiron/sqlx` with raw SQL queries, `$1`/`$2` placeholders
- **Migrations:** Go-embedded SQL files in `pkg/db/migrate/`, separate `.up.sql`/`.down.sql` per driver
- **Models:** `pkg/db/models/`
- **Transactions:** `db.TransactionContext` helper

## Configuration

Loaded from `$SOFT_SERVE_DATA_PATH/config.yaml` with env var overrides (`SOFT_SERVE_*` prefix). Parsed via `caarlos0/env`. Key config sections:

- `SSH` — listen addr, max timeout, key path
- `HTTP` — listen addr, TLS cert/key, public URL
- `Git` — listen addr, max connections, max timeout
- `Stats` — listen addr
- `LFS` — enabled flag
- `DB` — driver (sqlite/postgres), data source

## Metrics

Prometheus metrics throughout the codebase via `promauto`. Exposed at Stats server `/metrics`. Covers SSH connections, HTTP requests, git operations, TUI sessions, LFS transfers.

## Known TODOs

- `pkg/backend/backend.go` — proper caching interface (currently basic in-memory)
- `pkg/backend/lfs.go` — S3 storage support for LFS
- `pkg/lfs/ssh_client.go` — Git LFS SSH client (placeholder)
- `pkg/backend/hooks.go` — async hook execution
- `pkg/backend/user.go` — user repository ownership

## Common Pitfalls

1. **Never edit `*_templ.go` files** — they are generated. Edit `.templ` sources only.
2. **Context is king** — all dependencies flow through `context.Context`. Use `FromContext` accessors.
3. **Backend, not Store** — handlers call `Backend` methods, never `Store` directly.
4. **Git binary, not go-git** — push/pull shells out to `git`. `go-git` is used for read operations (log, tree, diff).
5. **Dual DB support** — SQL must work on both SQLite and PostgreSQL. Test with both if modifying queries.
6. **SSH commands are Cobra commands** — same pattern as CLI, dispatched in `CommandMiddleware`.
7. **Access levels gate everything** — check `AccessLevel` before any repo operation.

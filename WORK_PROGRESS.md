# Auto Poster — Work Progress Tracker

Track every task step-by-step. Check `[x]` when done.

Legend: `[x]` = done, `[ ]` = pending/in progress.

---

## Phase 0 — Project Setup

- [x] `mkdir -p cmd/server` — server entry point directory
- [x] `mkdir -p internal/{ai,research,content,scheduler,publisher,database}` — module skeletons
- [x] `mkdir -p migrations` — SQL migrations directory
- [x] `touch cmd/server/main.go` — placeholder (empty)
- [x] `touch .env` — placeholder (empty)
- [x] `touch docker-compose.yml` — placeholder (empty)
- [x] `touch README.md` — placeholder (empty)
- [x] `go mod init auto-poster` — go.mod created (`go 1.26.4`)
- [x] **MVP.md** — full technical doc written

## Phase 1 — Database (Step 2)

### Local PostgreSQL (Homebrew, running on :5432)

- [x] Create role `autoposter` (password `autoposter`)
  ```
  psql -h localhost -d postgres -c "CREATE ROLE autoposter LOGIN PASSWORD 'autoposter' CREATEDB;"
  ```
- [x] Create database `auto_poster` (owner `autoposter`)
  ```
  psql -h localhost -d postgres -c "CREATE DATABASE auto_poster OWNER autoposter;"
  ```
- [x] Verify connection:
  ```
  psql "postgres://autoposter:autoposter@localhost:5432/auto_poster" -c "SELECT 1;"
  ```
  → `SELECT 1` returned `1` ✓

### Docker fallback (only if local Postgres is not used)

- [ ] `docker-compose.yml` — postgres:17 container (contents documented in MVP.md)
- [ ] `docker compose up -d`
- [ ] `docker ps` — confirm `auto-poster-db` running

## Phase 2 — Go Driver (Step 3)

- [x] `go get github.com/jackc/pgx/v5` → v5.11.0 added
- [x] `go mod tidy`

## Phase 3 — Connection + Server (Step 4)

- [x] Write `internal/database/database.go` — pgxpool `Connect()` + ping
- [x] Write `cmd/server/main.go` — call `database.Connect()`, log start
- [x] `go run ./cmd/server`
- [x] Confirm output:
  ```
  Database connected successfully
  Auto Poster server started
  ```
  → ran successfully ✓

## Phase 4 — Schema (Step 5)

- [x] Write `internal/database/migrations/00001_create_posts.sql` — `posts` table (DDL in MVP.md)
- [x] Apply migration to `auto_poster` → table verified with `\d posts`
- [x] Add repository layer (`internal/database/posts.go` — CRUD: Create, GetByID, List, UpdateStatus, Delete)
- [x] Integrate **goose v3.28.0** migrations:
  - [x] `go get github.com/pressly/goose/v3`
  - [x] `//go:embed` migrations in `internal/database/database.go` (moved from root `migrations/`)
  - [x] `ApplyMigrations()` runs `provider.Up()` automatically at server start
  - [x] Added `-- +goose Up` / `-- +goose Down` annotations
  - [x] `goose_db_version` table tracks version 1 as applied
  - [x] Verified idempotent — 2nd run applies nothing

## Phase 5 — Environment & Config

- [x] Fill `.env` with `DATABASE_URL`
- [x] Add dotenv loading in `main.go` (godotenv v1.5.1)
- [x] Read `DATABASE_URL` from env in `database.go` (default = plan URL)

## Phase 6 — Module Stubs (later)

- [x] `internal/ai/` — `Provider` interface (`Generate(ctx, prompt)`) — provider abstraction
- [x] `internal/research/` — `Source` struct + `Service` interface (`Research(ctx, topic)`) — topic gathering
- [x] `internal/content/` — `Draft` struct + `Generator` interface (`Generate(ctx, topic, sources)`) — generation & templates
- [x] `internal/scheduler/` — `Scheduler` interface (`Schedule(ctx, post, at)`) — time-based scheduling
- [x] `internal/publisher/` — `Platform` consts + `Publisher` interface (`Publish(ctx, content)`) — platform API clients

## Phase 7 — Platform Posting (future)

- [ ] Connect first platform (Twitter/LinkedIn/blog)
- [ ] End-to-end: research → generate → schedule → publish
- [ ] Live posting test

---

## File Status Snapshot

| File/Path                      | Status   |
|--------------------------------|----------|
| `go.mod`                       | ✓ module created |
| `cmd/server/main.go`           | ✓ connects DB, loads .env, runs migrations |
| `internal/database/database.go`| ✓ pgxpool Connect + goose migrations |
| `internal/database/posts.go`   | ✓ CRUD repository |
| `internal/database/migrations/`| ✓ 00001_create_posts.sql (goose) |
| `.env`                         | ✓ DATABASE_URL set |
| `docker-compose.yml`           | empty — fallback only |
| `README.md`                    | empty |
| `MVP.md`                       | ✓ written |

## Verification Commands

```bash
go run ./cmd/server                          # run server
nc -z localhost 5432 && echo "pg open"        # is postgres up?
psql "postgres://autoposter:autoposter@localhost:5432/auto_poster" -c "SELECT 1;"
```
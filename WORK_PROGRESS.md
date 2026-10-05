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

- [x] `internal/ai/` — **Gemini provider** (`internal/ai/gemini.go`, OpenAI-compatible endpoint, model `gemini-3.8-flash`, retry on 429/503). Note: `gemini-2.0-flash` is deprecated by Google.
- [x] `internal/research/` — `Source` struct + `Service` interface (`Research(ctx, topic)`) — topic gathering
- [x] `internal/content/` — `Draft` struct + `Generator` interface (`Generate(ctx, topic, sources)`) — generation & templates
- [x] `internal/scheduler/` — `Scheduler` interface (`Schedule(ctx, post, at)`) — time-based scheduling
- [x] `internal/publisher/` — `Platform` consts + `Publisher` interface (`Publish(ctx, content)`) — platform API clients

## Phase 7 — Platform Posting (future)

### Portfolio blog API (Django side — `/Volumes/New Volume/Development/web/portfolio-`)

- [x] `portfolio_project/settings.py` — added `BLOG_API_TOKEN` (env)
- [x] `core/api.py` — `blog_api_create` view: `POST /api/blog/posts/`, Bearer token auth, JSON in/out (201/400/401)
- [x] `core/urls.py` — registered `api/blog/posts/` route
- [x] Tested with curl: 401 (no/wrong token), 400 (missing title), 201 (valid post, slug auto-generated), live on `/blog/<slug>/`

### Auto-poster publisher (Go side)

- [x] Migration `00002_add_posts_title.sql` — added `title` column to `posts`
- [x] `internal/database/posts.go` — `Post.Title` + `Create(title, ...)`
- [x] `internal/publisher/publisher.go` — `PublishRequest{Title, Content, Category, ShortDescription}` + `Publish() (string, error)`
- [x] `internal/publisher/djangoblog.go` — `DjangoBlogPublisher` (POST `/api/blog/posts/`, Bearer token)
- [x] `.env` — `BLOG_API_URL`, `BLOG_API_TOKEN`
- [x] `cmd/publish-smoke/main.go` — E2E smoke test CLI
- [x] E2E verified: Go publisher → Django → blog live at `/blog/go-auto-poster-smoke-test/` (test post deleted after)

### Next (not done)

- [x] **LIVE deployment** — full pipeline now works on production (see below)
- [ ] End-to-end pipeline: research → generate → schedule → publish
- [ ] Wire scheduler to actually publish from `posts` table (status → published)

## Phase 8 — LIVE Deployment (free: GitHub Actions + Supabase + Vercel)

### Infrastructure
- [x] **Supabase** free Postgres — `DATABASE_URL` (production DB, migrations auto-applied by `publish-due`)
- [x] **GitHub Actions** cron (`.github/workflows/publish.yml`, `*/15 * * * *` = every 15 min) runs `cmd/publish-due` + manual `workflow_dispatch`. Worker only publishes inside the dashboard-configured window (daily start time + interval).
- [x] **Reliable trigger** — GitHub throttles `schedule` crons on low-activity repos (observed ~every 3h, not 15m). A Supabase `pg_cron` + `pg_net` job (`scripts/setup_supabase_cron.sh`) now calls the workflow_dispatch API every 15 min instead. The token is stored in Supabase Vault (`auto_poster_github_pat`).
- [x] **Vercel** free — portfolio live at `https://www.muntasirashif.com`, blog API + settings API deployed
- [x] Secrets set: `DATABASE_URL`, `BLOG_API_URL=https://www.muntasirashif.com`, `BLOG_API_TOKEN`
- [x] Scheduler: `internal/scheduler` — `ProcessDue` (shared) + `Worker` (local always-on); `cmd/publish-due` (cron one-shot)
- [x] `cmd/schedule` CLI — insert a scheduled post
- [x] pgx `QueryExecModeDescribeExec` — works with Supabase PgBouncer pooler (6543)
- [x] Verified: `gh workflow run` published a live post → `/blog/auto-poster-live-test/` (200) + `posts.status='published'`

### Dashboard Auto Poster options (Django)
- [x] `AutoPosterOption` model + migration `0002` (`enabled`, `blog_api_url`, `default_category`, `content_footer`)
- [x] Settings page: `/dashboard/autoposter/settings/` (+ sidebar "Auto Poster" link + button on Blog Posts page)
- [x] `GET /api/autoposter/settings/` (Bearer auth) — served to the worker
- [x] Go respects options: `enabled` (skip if off), `default_category`, `content_footer`
- [x] E2E verified: category "tech" + footer appended to published post
- [x] **Post start time + interval + timezone** (migration `0003`): `daily_publish_time`, `schedule_interval_minutes`, `timezone` editable in dashboard
- [x] Worker `PublishAllowed()` — never publishes before the daily start time; interval enforced as a minimum gap since the last publish (robust to delayed/throttled cron runs)
- [x] **Scheduled Posts queue** (migration `0003`, shared `posts` table): `/dashboard/posts/scheduled/` list + add/edit/delete — full dashboard control, nothing publishes before its scheduled time
- [x] Tested: list 200, create → status=scheduled, edit 200, delete works (production Supabase DB)
- [x] **AI generation** — `cmd/generate -topic "..." [-publish]`: Gemini writes a full HTML blog post, queued in `posts` (+ optional immediate publish). Verified live: generated post published to the blog with category + footer.
- [x] **AI cover image** — `cmd/generate -image`: **AI Horde** (free, anonymous) generates a cover image; stored as `posts.image_url` (migration `00004`); blog API accepts `image_base64` (JSON) → saved to Cloudinary; `publish-due` downloads + attaches at publish time. Verified live with cover image.
- [x] Fixed blog API `DataError` — truncated + deduped slug (was exceeding `SlugField(50)` for long titles)
- [x] Multipart uploads hang on Vercel serverless → image sent as base64 JSON instead

---

## File Status Snapshot

| File/Path                      | Status   |
|--------------------------------|----------|
| `go.mod`                       | ✓ module created |
| `cmd/server/main.go`           | ✓ connects DB, loads .env, runs migrations |
| `internal/database/database.go`| ✓ pgxpool Connect + goose migrations (pooler-safe) |
| `internal/database/posts.go`   | ✓ CRUD repository + ListDue |
| `internal/database/migrations/`| ✓ 00001_create_posts.sql + 00002_add_posts_title.sql (goose) |
| `internal/publisher/`         | ✓ Publisher interface + DjangoBlogPublisher + AutoPosterSettings |
| `internal/scheduler/`         | ✓ ProcessDue + Worker |
| `cmd/publish-due/`            | ✓ cron one-shot (production worker) |
| `cmd/schedule/`               | ✓ schedule a post CLI |
| `.github/workflows/publish.yml`| ✓ GitHub Actions cron (5 min) |
| `.env`                         | ✓ DATABASE_URL, BLOG_API_URL, BLOG_API_TOKEN |
| `docker-compose.yml`           | empty — fallback only |
| `README.md`                    | empty |
| `MVP.md`                       | ✓ written |

## Verification Commands

```bash
go run ./cmd/server                          # run server
nc -z localhost 5432 && echo "pg open"        # is postgres up?
psql "postgres://autoposter:autoposter@localhost:5432/auto_poster" -c "SELECT 1;"
```
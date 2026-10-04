# Auto Poster — MVP Technical Doc

## 1. Overview

Auto Poster is a Go-based system that automates the content lifecycle: **research → generate → schedule → publish** across social platforms.

This MVP establishes the solid foundation first: **Go application + PostgreSQL connection**. AI, web research, content generation, scheduler, and publisher modules exist as package skeletons and are wired in later phases.

## 2. Tech Stack

| Layer      | Technology                          |
|------------|-------------------------------------|
| Language   | Go 1.26.4                           |
| Driver     | `github.com/jackc/pgx/v5` (pgxpool) |
| Database   | PostgreSQL 17                       |
| Migration  | `github.com/pressly/goose/v3` (embedded, auto-run on start) |
| Containers | Docker Compose (`postgres:17`) — optional fallback |

## 3. Prerequisites

- Go ≥ 1.26 (`go version`)
- PostgreSQL 17 running locally (Homebrew) **OR** Docker with `docker compose`

Check: `nc -z localhost 5432 && echo open` → port 5432 must be open.

## 4. Project Structure

```
auto-poster/
├── cmd/
│   └── server/
│       └── main.go            # Entry point: connect DB, start server
├── internal/
│   ├── ai/                    # (stub) AI provider integrations (text/image gen)
│   ├── research/              # (stub) web research & topic gathering
│   ├── content/               # (stub) content generation & templating
│   ├── scheduler/             # (stub) scheduling engine (cron/queued posts)
│   ├── publisher/             # (stub) platform API clients (posting)
│   └── database/
│       ├── database.go        # pgxpool connection + goose migrations
│       ├── posts.go           # posts CRUD repository
│       └── migrations/        # embedded SQL migrations (00001_create_posts.sql)
├── .env                       # local environment variables
├── docker-compose.yml         # optional PostgreSQL container
├── go.mod                     # module: auto-poster
├── MVP.md                     # this document
└── README.md
```

## 5. Database Setup

### 5.1 Local PostgreSQL (primary)

Create role + database (matching the plan's connection string):

```sql
CREATE ROLE autoposter LOGIN PASSWORD 'autoposter' CREATEDB;
CREATE DATABASE auto_poster OWNER autoposter;
```

Run with the local superuser (default = macOS username):

```bash
psql -h localhost -d postgres -c "CREATE ROLE autoposter LOGIN PASSWORD 'autoposter' CREATEDB;"
psql -h localhost -d postgres -c "CREATE DATABASE auto_poster OWNER autoposter;"
```

Verify: `psql "postgres://autoposter:autoposter@localhost:5432/auto_poster" -c "SELECT 1;"`

### 5.2 Docker fallback

`docker-compose.yml`:

```yaml
services:
  postgres:
    image: postgres:17
    container_name: auto-poster-db
    restart: unless-stopped
    environment:
      POSTGRES_USER: autoposter
      POSTGRES_PASSWORD: autoposter
      POSTGRES_DB: auto_poster
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

```bash
docker compose up -d
docker ps   # expect: auto-poster-db / postgres:17 / 5432->5432
```

## 6. Schema (MVP Migrations)

Migrations live in `internal/database/migrations/` and are **managed by goose** — embedded via `//go:embed` and auto-applied on server start (`ApplyMigrations` in `database.go`). Versions tracked in the `goose_db_version` table.

Phase 1 adds the `posts` table (`00001_create_posts.sql`):

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS posts (
    id           BIGSERIAL PRIMARY KEY,
    platform     TEXT        NOT NULL,          -- e.g. twitter, linkedin, blog
    content      TEXT        NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'draft', -- draft | scheduled | published | failed
    scheduled_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS posts;
```

New migrations = next numbered file (`00002_*.sql`), each with `-- +goose Up` / `-- +goose Down` blocks.

Planned later phases:

- `sources` — research feeds/keywords per topic
- `accounts` — connected platform accounts + tokens (secrets never committed)

## 7. Connection Layer

`internal/database/database.go`:

```go
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

const defaultDatabaseURL = "postgres://autoposter:autoposter@localhost:5432/auto_poster"

//go:embed migrations/*.sql
var embedMigrations embed.FS

func databaseURL() string {
	if u := os.Getenv("DATABASE_URL"); u != "" {
		return u
	}
	return defaultDatabaseURL
}

func Connect() (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL())
	if err != nil {
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	log.Println("Database connected successfully")

	return pool, nil
}

func ApplyMigrations() error {
	db, err := sql.Open("pgx", databaseURL())
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return err
	}

	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		return err
	}

	results, err := provider.Up(context.Background())
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	for _, res := range results {
		log.Printf("migration applied: %s", res.Source.Path)
	}

	return nil
}
```

`cmd/server/main.go`:

```go
package main

import (
	"log"

	"github.com/joho/godotenv"

	"auto-poster/internal/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if err := database.ApplyMigrations(); err != nil {
		log.Fatal(err)
	}

	log.Println("Auto Poster server started")
}
```

## 8. Module Architecture

| Package          | Responsibility (MVP = stub)                        |
|------------------|-----------------------------------------------------|
| `internal/ai`    | AI provider abstraction (generate text/images)      |
| `internal/research` | Gather topics/sources from the web               |
| `internal/content`  | Compose + format content from research           |
| `internal/scheduler` | Schedule posts (time-based queue, cron)         |
| `internal/publisher` | Send posts to platform APIs                      |
| `internal/database` | Pool, migrations, repositories                   |

Data flow (target): `research → ai/content → scheduler → publisher → posts` table.

## 9. Environment Variables

| Variable      | Default                                          | Purpose                 |
|---------------|--------------------------------------------------|-------------------------|
| `DATABASE_URL`| `postgres://autoposter:autoposter@localhost:5432/auto_poster` | Postgres connection |

`.env` lives at project root and is loaded by `godotenv` in `main.go` before connecting.

## 10. Run Instructions

```bash
# 1. dependencies
go get github.com/jackc/pgx/v5 github.com/pressly/goose/v3
go mod tidy

# 2. run server (connects DB + auto-applies goose migrations)
go run ./cmd/server
```

Expected output (first run):

```
Database connected successfully
migration applied: 00001_create_posts.sql
Auto Poster server started
```

Subsequent runs apply no migrations (idempotent).

## 11. MVP Scope & Non-Goals

**In scope (Phase 1):**
- Go module + PostgreSQL 17 connection (pgxpool, ping)
- `posts` table migration
- Server entry point

**Deferred (later phases):**
- AI generation, web research, content templates
- Scheduling engine
- Platform publishers (Twitter/LinkedIn/blog APIs)
- Auth, users, API endpoints

## 12. Roadmap

| Milestone | Deliverable                                      |
|-----------|--------------------------------------------------|
| M1        | Go + PostgreSQL connection solid (this MVP)      |
| M2        | `posts` migration + repository layer             |
| M3        | Research module (web sources)                    |
| M4        | AI + content generation                          |
| M5        | Scheduler + publisher wiring                     |
| M6        | Live posting to a platform                       |

## 13. Debugging

- **`FATAL: role "autoposter" does not exist`** → run the `CREATE ROLE` step
- **`FATAL: database "auto_poster" does not exist`** → run the `CREATE DATABASE` step
- **`connection refused` on 5432** → start PostgreSQL (`brew services start postgresql@17` or `docker compose up -d`)
- **Port 5432 already in use** → Homebrew Postgres conflicts with Docker; pick one (local is primary here)
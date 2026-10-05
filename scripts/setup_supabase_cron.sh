#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# Wires up a Supabase pg_cron job that dispatches the auto-poster GitHub
# Actions workflow every 15 minutes. This works around GitHub throttling the
# workflow's own `schedule` cron.
#
# Usage:
#   GITHUB_PAT=github_pat_xxx \
#   SUPABASE_DB_URL="postgres://postgres.<ref>:<password>@<host>:5432/postgres" \
#   ./scripts/setup_supabase_cron.sh
#
# Create the token at https://github.com/settings/personal-access-tokens/new
#   Repository access : Only select repositories -> MuntasirAsif/auto-poster
#   Permissions       : Actions -> Read and write
#
# SUPABASE_DB_URL defaults to $DATABASE_URL when not set. Use the direct
# (non-pooling) connection string for extension setup.
# ---------------------------------------------------------------------------

: "${GITHUB_PAT:?set GITHUB_PAT to a fine-grained GitHub token with 'Actions: Read and write'}"

DB_URL="${SUPABASE_DB_URL:-${DATABASE_URL:-}}"
: "${DB_URL:?set SUPABASE_DB_URL (or DATABASE_URL) to your Supabase Postgres URL}"

# Strip Supabase pooler query params that psql does not understand.
DB_URL="${DB_URL%%\&supa=*}"
DB_URL="${DB_URL%%\?supa=*}"

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v psql >/dev/null 2>&1; then
  echo "psql not found. Install postgresql-client, or run scripts/supabase_pg_cron.sql" >&2
  echo "in the Supabase SQL Editor (replace :'github_pat' with your quoted token)." >&2
  exit 1
fi

echo "Installing pg_cron/pg_net job 'auto-poster-dispatch'..."
psql "$DB_URL" -v ON_ERROR_STOP=1 -v github_pat="$GITHUB_PAT" -f "$DIR/supabase_pg_cron.sql"

echo
echo "Done. Verify with:"
echo "  select jobname, schedule, active from cron.job where jobname = 'auto-poster-dispatch';"

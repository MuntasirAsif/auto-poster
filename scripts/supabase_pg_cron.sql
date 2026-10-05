-- ============================================================================
-- Reliable auto-poster scheduler (Supabase)
-- ----------------------------------------------------------------------------
-- GitHub throttles `schedule` crons on low-activity repositories, so the
-- workflow's own cron is unreliable. This installs a pg_cron job that calls
-- the GitHub Actions workflow_dispatch API directly.
--
-- It dispatches `.github/workflows/daily.yml` once per day at 19:00
-- Asia/Dhaka (13:00 UTC), which generates and publishes 3 posts.
--
-- Prerequisites (Supabase Dashboard > Database > Extensions):
--   pg_cron, pg_net and vault must be available.
--
-- Run via scripts/setup_supabase_cron.sh (preferred) or paste into the
-- Supabase SQL Editor, replacing :'github_pat' with a quoted token literal,
-- e.g. 'github_pat_xxxxxxxx'.
-- ============================================================================

create extension if not exists pg_cron;
create extension if not exists pg_net;

-- Store (or replace) the GitHub token in Vault.
delete from vault.secrets where name = 'auto_poster_github_pat';
select vault.create_secret(
  :'github_pat',
  'auto_poster_github_pat',
  'Fine-grained PAT used to dispatch the auto-poster GitHub Actions workflow'
);

-- Drop any previous dispatch job, then recreate it.
select cron.unschedule(jobid) from cron.job where jobname = 'auto-poster-dispatch';

select cron.schedule(
  'auto-poster-dispatch',
  '0 13 * * *',
  $job$
  select net.http_post(
    url := 'https://api.github.com/repos/MuntasirAsif/auto-poster/actions/workflows/daily.yml/dispatches',
    headers := jsonb_build_object(
      'Authorization', 'Bearer ' || (
        select decrypted_secret
        from vault.decrypted_secrets
        where name = 'auto_poster_github_pat'
        limit 1
      ),
      'Accept', 'application/vnd.github+json',
      'X-GitHub-Api-Version', '2022-11-28',
      'Content-Type', 'application/json'
    ),
    body := '{"ref":"main"}'::jsonb,
    timeout_milliseconds := 10000
  );
  $job$
);

-- Verify:
--   select jobname, schedule, active from cron.job where jobname = 'auto-poster-dispatch';
--   select id, status_code, content from net._http_response order by id desc limit 5;

-- Runs once on first start of each dev container.
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE EXTENSION IF NOT EXISTS pg_stat_kcache;
CREATE EXTENSION IF NOT EXISTS pg_wait_sampling;
CREATE EXTENSION IF NOT EXISTS hypopg;

-- Least-privilege monitoring role, matching docs/onboarding-and-deploy.md.
CREATE ROLE tuskline_monitor LOGIN PASSWORD 'tuskline_monitor' CONNECTION LIMIT 5;
GRANT pg_monitor TO tuskline_monitor;
ALTER ROLE tuskline_monitor SET default_transaction_read_only = on;
ALTER ROLE tuskline_monitor SET statement_timeout = '5s';
ALTER ROLE tuskline_monitor SET lock_timeout = '1s';
ALTER ROLE tuskline_monitor SET idle_in_transaction_session_timeout = '10s';

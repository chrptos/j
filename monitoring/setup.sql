\getenv metrics_password METRICS_PASSWORD
BEGIN;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
SELECT 'CREATE ROLE j_metrics LOGIN' WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='j_metrics') \gexec
SELECT format('ALTER ROLE j_metrics PASSWORD %L', :'metrics_password') \gexec
GRANT pg_monitor TO j_metrics;
SELECT format('GRANT CONNECT ON DATABASE %I TO j_metrics',current_database()) \gexec
COMMIT;

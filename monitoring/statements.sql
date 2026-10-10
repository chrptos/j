SELECT queryid,calls,total_exec_time,rows,shared_blks_hit,shared_blks_read
FROM pg_stat_statements
WHERE dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
  AND userid=(SELECT oid FROM pg_roles WHERE rolname=current_user)
ORDER BY total_exec_time DESC;

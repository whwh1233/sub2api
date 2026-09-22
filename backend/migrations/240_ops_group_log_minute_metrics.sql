-- Separate log-derived counts from the previous ingress-event definition.
-- Preserve old data; never relabel Redis-derived historical counts as logs.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';
CREATE TABLE IF NOT EXISTS ops_group_log_minute_metrics
    (LIKE ops_group_minute_metrics INCLUDING ALL);
COMMENT ON TABLE ops_group_log_minute_metrics IS
    'PostgreSQL usage/error log rollups. group_id=-1 marks scanned minutes. started_count retains the API name but counts completed log records, including explicit 499 cancellations.';

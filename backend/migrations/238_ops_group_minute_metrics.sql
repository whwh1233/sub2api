-- Durable history for the realtime entry-group counters. No FK: retain history
-- after a group is deleted, and allow group 0 (unassigned).
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '5min';

CREATE TABLE IF NOT EXISTS ops_group_minute_metrics (
    bucket_start TIMESTAMPTZ NOT NULL,
    group_id BIGINT NOT NULL CHECK (group_id >= -1),
    group_name TEXT NOT NULL DEFAULT '',
    platform VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT '',
    started_count BIGINT NOT NULL DEFAULT 0 CHECK (started_count >= 0),
    success_count BIGINT NOT NULL DEFAULT 0 CHECK (success_count >= 0),
    failed_count BIGINT NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
    cancelled_count BIGINT NOT NULL DEFAULT 0 CHECK (cancelled_count >= 0),
    partial BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bucket_start, group_id),
    CHECK (EXTRACT(SECOND FROM bucket_start) = 0),
    CHECK (group_id <> -1 OR (started_count = 0 AND success_count = 0 AND failed_count = 0 AND cancelled_count = 0))
);
CREATE INDEX IF NOT EXISTS idx_ops_group_minute_metrics_group_time
    ON ops_group_minute_metrics (group_id, bucket_start DESC);
COMMENT ON TABLE ops_group_minute_metrics IS
    'Minute counters for group realtime history; group_id=-1 is a coverage marker, 0 is unassigned. No automatic retention cleanup.';
COMMENT ON COLUMN ops_group_minute_metrics.partial IS
    'Incomplete collection: excluded from rates and RPM averages. Absence of a coverage marker also means unknown, never zero traffic.';

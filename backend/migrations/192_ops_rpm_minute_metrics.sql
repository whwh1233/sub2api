-- Admin-only RPM trend rollups.
--
-- The minute collector writes completed one-minute buckets from usage_logs and
-- ops_error_logs. Coarser buckets are derived from these rows so dashboard
-- reads never need to scan the high-volume request tables.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '5min';

CREATE TABLE IF NOT EXISTS ops_rpm_metrics (
    id BIGSERIAL PRIMARY KEY,
    bucket_start TIMESTAMPTZ NOT NULL,
    bucket_seconds INT NOT NULL,
    dimension_type VARCHAR(24) NOT NULL,
    dimension_key VARCHAR(255) NOT NULL,
    dimension_label TEXT NOT NULL,
    success_count BIGINT NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ops_rpm_metrics_bucket_seconds_check
        CHECK (bucket_seconds IN (60, 300, 7200)),
    CONSTRAINT ops_rpm_metrics_dimension_type_check
        CHECK (dimension_type IN ('platform', 'model', 'account', 'user')),
    CONSTRAINT ops_rpm_metrics_counts_check
        CHECK (success_count >= 0 AND error_count >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_rpm_metrics_unique_bucket_dimension
    ON ops_rpm_metrics (bucket_seconds, bucket_start, dimension_type, dimension_key);

CREATE INDEX IF NOT EXISTS idx_ops_rpm_metrics_dimension_time
    ON ops_rpm_metrics (bucket_seconds, dimension_type, bucket_start DESC, dimension_key);

CREATE INDEX IF NOT EXISTS idx_ops_rpm_metrics_bucket_time
    ON ops_rpm_metrics (bucket_seconds, bucket_start DESC);

COMMENT ON TABLE ops_rpm_metrics IS
    'Admin-only RPM time-series rollups by platform, requested model, account, and user.';

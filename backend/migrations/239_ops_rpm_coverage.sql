-- Coverage is committed with counters, including minutes with no requests.
-- Existing rollups cannot prove coverage and are intentionally not backfilled.
SET LOCAL lock_timeout = '5s';
CREATE TABLE IF NOT EXISTS ops_rpm_coverage (
    bucket_seconds INT NOT NULL CHECK (bucket_seconds IN (60, 300, 7200)),
    bucket_start TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (bucket_seconds, bucket_start)
);

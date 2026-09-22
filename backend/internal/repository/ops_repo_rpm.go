package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) UpsertRPMMinuteMetrics(ctx context.Context, startTime, endTime time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if !startTime.Before(endTime) {
		return nil
	}

	const q = `
WITH usage_base AS MATERIALIZED (
  SELECT
    date_trunc('minute', ul.created_at) AS bucket_start,
    COALESCE(NULLIF(a.platform, ''), 'unknown') AS platform_key,
    COALESCE(NULLIF(ul.requested_model, ''), NULLIF(ul.model, ''), 'unknown') AS model_key,
    ul.account_id::text AS account_key,
    COALESCE(NULLIF(a.name, ''), '#' || ul.account_id::text) AS account_label,
    ul.user_id::text AS user_key,
    COALESCE(NULLIF(u.username, ''), NULLIF(u.email, ''), '#' || ul.user_id::text) AS user_label
  FROM usage_logs ul
  LEFT JOIN accounts a ON a.id = ul.account_id
  LEFT JOIN users u ON u.id = ul.user_id
  WHERE ul.created_at >= $1 AND ul.created_at < $2
),
error_base AS MATERIALIZED (
  SELECT
    date_trunc('minute', o.created_at) AS bucket_start,
    COALESCE(NULLIF(o.platform, ''), NULLIF(a.platform, ''), 'unknown') AS platform_key,
    COALESCE(NULLIF(o.requested_model, ''), NULLIF(o.model, ''), 'unknown') AS model_key,
    o.account_id::text AS account_key,
    CASE WHEN o.account_id IS NULL THEN '' ELSE COALESCE(NULLIF(a.name, ''), '#' || o.account_id::text) END AS account_label,
    o.user_id::text AS user_key,
    CASE WHEN o.user_id IS NULL THEN '' ELSE COALESCE(NULLIF(u.username, ''), NULLIF(u.email, ''), '#' || o.user_id::text) END AS user_label
  FROM ops_error_logs o
  LEFT JOIN accounts a ON a.id = o.account_id
  LEFT JOIN users u ON u.id = o.user_id
  WHERE o.created_at >= $1 AND o.created_at < $2
    AND COALESCE(o.status_code, 0) >= 400
    AND o.is_count_tokens = FALSE
),
dimension_rows AS (
  SELECT bucket_start, 'platform'::text AS dimension_type, platform_key AS dimension_key, platform_key AS dimension_label, 1::bigint AS success_count, 0::bigint AS error_count FROM usage_base
  UNION ALL
  SELECT bucket_start, 'model', model_key, model_key, 1, 0 FROM usage_base
  UNION ALL
  SELECT bucket_start, 'account', account_key, account_label, 1, 0 FROM usage_base WHERE account_key IS NOT NULL AND account_key <> ''
  UNION ALL
  SELECT bucket_start, 'user', user_key, user_label, 1, 0 FROM usage_base WHERE user_key IS NOT NULL AND user_key <> ''
  UNION ALL
  SELECT bucket_start, 'platform', platform_key, platform_key, 0, 1 FROM error_base
  UNION ALL
  SELECT bucket_start, 'model', model_key, model_key, 0, 1 FROM error_base
  UNION ALL
  SELECT bucket_start, 'account', account_key, account_label, 0, 1 FROM error_base WHERE account_key IS NOT NULL AND account_key <> ''
  UNION ALL
  SELECT bucket_start, 'user', user_key, user_label, 0, 1 FROM error_base WHERE user_key IS NOT NULL AND user_key <> ''
),
aggregated AS (
  SELECT
    bucket_start,
    dimension_type,
    dimension_key,
    MAX(dimension_label) AS dimension_label,
    SUM(success_count) AS success_count,
    SUM(error_count) AS error_count
  FROM dimension_rows
  GROUP BY bucket_start, dimension_type, dimension_key
)
INSERT INTO ops_rpm_metrics (
  bucket_start, bucket_seconds, dimension_type, dimension_key, dimension_label,
  success_count, error_count, computed_at
)
SELECT bucket_start, 60, dimension_type, dimension_key, dimension_label,
       success_count, error_count, NOW()
FROM aggregated
ON CONFLICT (bucket_seconds, bucket_start, dimension_type, dimension_key)
DO UPDATE SET
  dimension_label = EXCLUDED.dimension_label,
  success_count = EXCLUDED.success_count,
  error_count = EXCLUDED.error_count,
  computed_at = NOW()`

	_, err := r.db.ExecContext(ctx, q, startTime.UTC(), endTime.UTC())
	return err
}

func (r *opsRepository) UpsertRPMRollup(
	ctx context.Context,
	sourceBucketSeconds, targetBucketSeconds int,
	startTime, endTime time.Time,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if !startTime.Before(endTime) {
		return nil
	}
	if !((sourceBucketSeconds == 60 && targetBucketSeconds == 300) ||
		(sourceBucketSeconds == 300 && targetBucketSeconds == 7200)) {
		return fmt.Errorf("unsupported RPM rollup %d -> %d", sourceBucketSeconds, targetBucketSeconds)
	}

	const q = `
INSERT INTO ops_rpm_metrics (
  bucket_start, bucket_seconds, dimension_type, dimension_key, dimension_label,
  success_count, error_count, computed_at
)
SELECT
  date_bin(make_interval(secs => $2), bucket_start, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
  $2,
  dimension_type,
  dimension_key,
  MAX(dimension_label),
  SUM(success_count),
  SUM(error_count),
  NOW()
FROM ops_rpm_metrics
WHERE bucket_seconds = $1
  AND bucket_start >= $3 AND bucket_start < $4
GROUP BY 1, 3, 4
ON CONFLICT (bucket_seconds, bucket_start, dimension_type, dimension_key)
DO UPDATE SET
  dimension_label = EXCLUDED.dimension_label,
  success_count = EXCLUDED.success_count,
  error_count = EXCLUDED.error_count,
  computed_at = NOW()`

	_, err := r.db.ExecContext(ctx, q, sourceBucketSeconds, targetBucketSeconds, startTime.UTC(), endTime.UTC())
	return err
}

func (r *opsRepository) CleanupRPMMetrics(
	ctx context.Context,
	minuteCutoff, fiveMinuteCutoff, twoHourCutoff time.Time,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	const q = `
DELETE FROM ops_rpm_metrics
WHERE (bucket_seconds = 60 AND bucket_start < $1)
   OR (bucket_seconds = 300 AND bucket_start < $2)
   OR (bucket_seconds = 7200 AND bucket_start < $3)`
	_, err := r.db.ExecContext(ctx, q, minuteCutoff.UTC(), fiveMinuteCutoff.UTC(), twoHourCutoff.UTC())
	return err
}

func (r *opsRepository) GetRPMTrend(ctx context.Context, filter *service.OpsRPMTrendFilter) (*service.OpsRPMTrendResponse, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		return nil, fmt.Errorf("nil RPM trend filter")
	}
	if filter.SourceBucketSeconds <= 0 || filter.OutputBucketSeconds <= 0 || filter.TopN <= 0 {
		return nil, fmt.Errorf("invalid RPM trend filter")
	}

	const q = `
WITH bucketed AS MATERIALIZED (
  SELECT
    date_bin(make_interval(secs => $5), bucket_start, TIMESTAMPTZ '1970-01-01 00:00:00+00') AS display_bucket,
    dimension_key,
    MAX(dimension_label) AS dimension_label,
    SUM(success_count) AS success_count,
    SUM(error_count) AS error_count
  FROM ops_rpm_metrics
  WHERE bucket_seconds = $1
    AND dimension_type = $2
    AND bucket_start >= $3 AND bucket_start < $4
  GROUP BY 1, 2
),
ranked AS (
  SELECT dimension_key, MAX(dimension_label) AS dimension_label,
         SUM(success_count + error_count) AS total_count,
         ROW_NUMBER() OVER (
           ORDER BY SUM(success_count + error_count) DESC, dimension_key ASC
         ) AS series_rank
  FROM bucketed
  GROUP BY dimension_key
),
combined AS (
  SELECT b.display_bucket, b.dimension_key, r.dimension_label, r.series_rank,
         b.success_count, b.error_count
  FROM bucketed b
  JOIN ranked r USING (dimension_key)
  WHERE r.series_rank <= $6
  UNION ALL
  SELECT b.display_bucket, '__other__', 'Other', $6 + 1,
         SUM(b.success_count), SUM(b.error_count)
  FROM bucketed b
  JOIN ranked r USING (dimension_key)
  WHERE r.series_rank > $6
  GROUP BY b.display_bucket
  HAVING SUM(b.success_count + b.error_count) > 0
)
SELECT display_bucket, dimension_key, dimension_label, success_count, error_count
FROM combined
ORDER BY series_rank, dimension_key, display_bucket`

	rows, err := r.db.QueryContext(
		ctx,
		q,
		filter.SourceBucketSeconds,
		string(filter.Dimension),
		filter.StartTime.UTC(),
		filter.EndTime.UTC(),
		filter.OutputBucketSeconds,
		filter.TopN,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	seriesByKey := make(map[string]*service.OpsRPMTrendSeries)
	seriesOrder := make([]string, 0, filter.TopN+1)
	minutes := float64(filter.OutputBucketSeconds) / 60
	if minutes <= 0 {
		minutes = 1
	}
	for rows.Next() {
		var bucket time.Time
		var key, label string
		var successCount, errorCount int64
		if err := rows.Scan(&bucket, &key, &label, &successCount, &errorCount); err != nil {
			return nil, err
		}
		series := seriesByKey[key]
		if series == nil {
			series = &service.OpsRPMTrendSeries{Key: key, Label: label, Points: make([]*service.OpsRPMTrendPoint, 0)}
			seriesByKey[key] = series
			seriesOrder = append(seriesOrder, key)
		}
		series.Points = append(series.Points, &service.OpsRPMTrendPoint{
			BucketStart:  bucket.UTC(),
			SuccessCount: successCount,
			ErrorCount:   errorCount,
			SuccessRPM:   float64(successCount) / minutes,
			ErrorRPM:     float64(errorCount) / minutes,
			TotalRPM:     float64(successCount+errorCount) / minutes,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &service.OpsRPMTrendResponse{Series: make([]*service.OpsRPMTrendSeries, 0, len(seriesOrder))}
	bucketDuration := time.Duration(filter.OutputBucketSeconds) * time.Second
	firstBucket := time.Unix(
		(filter.StartTime.UTC().Unix()/int64(filter.OutputBucketSeconds))*int64(filter.OutputBucketSeconds),
		0,
	).UTC()
	for _, key := range seriesOrder {
		series := seriesByKey[key]
		pointsByBucket := make(map[int64]*service.OpsRPMTrendPoint, len(series.Points))
		for _, point := range series.Points {
			pointsByBucket[point.BucketStart.Unix()] = point
		}
		filled := make([]*service.OpsRPMTrendPoint, 0, int(filter.EndTime.Sub(firstBucket)/bucketDuration)+1)
		for bucket := firstBucket; bucket.Before(filter.EndTime); bucket = bucket.Add(bucketDuration) {
			if point := pointsByBucket[bucket.Unix()]; point != nil {
				filled = append(filled, point)
				continue
			}
			filled = append(filled, &service.OpsRPMTrendPoint{BucketStart: bucket})
		}
		series.Points = filled
		result.Series = append(result.Series, series)
	}

	var latest sql.NullTime
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT MAX(bucket_start) FROM ops_rpm_metrics WHERE bucket_seconds = $1 AND bucket_start >= $2 AND bucket_start < $3`,
		filter.SourceBucketSeconds,
		filter.StartTime.UTC(),
		filter.EndTime.UTC(),
	).Scan(&latest); err != nil {
		return nil, err
	}
	if latest.Valid {
		completeThrough := latest.Time.UTC().Add(time.Duration(filter.SourceBucketSeconds) * time.Second)
		result.CompleteThrough = &completeThrough
	}
	return result, nil
}

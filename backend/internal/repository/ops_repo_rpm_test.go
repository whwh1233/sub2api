package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryUpsertRPMMinuteMetricsUsesBoundedRequestLogs(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Minute)

	mock.ExpectExec(`(?s)WITH usage_base AS MATERIALIZED.*FROM usage_logs ul.*ul.created_at >= \$1 AND ul.created_at < \$2.*error_base AS MATERIALIZED.*FROM ops_error_logs o.*o.is_count_tokens = FALSE.*ON CONFLICT`).
		WithArgs(start, end).
		WillReturnResult(sqlmock.NewResult(0, 24))

	require.NoError(t, repo.UpsertRPMMinuteMetrics(context.Background(), start, end))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryGetRPMTrendFillsIdleBucketsAndKeepsRankOrder(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}
	start := time.Date(2026, 8, 5, 8, 0, 30, 0, time.UTC)
	end := time.Date(2026, 8, 5, 8, 3, 0, 0, time.UTC)
	filter := &service.OpsRPMTrendFilter{
		StartTime: start, EndTime: end,
		Dimension:           service.OpsRPMDimensionPlatform,
		SourceBucketSeconds: 60, OutputBucketSeconds: 60, TopN: 10,
	}

	rows := sqlmock.NewRows([]string{
		"display_bucket", "dimension_key", "dimension_label", "success_count", "error_count",
	}).
		AddRow(time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC), "openai", "openai", int64(12), int64(1)).
		AddRow(time.Date(2026, 8, 5, 8, 2, 0, 0, time.UTC), "openai", "openai", int64(6), int64(0)).
		AddRow(time.Date(2026, 8, 5, 8, 1, 0, 0, time.UTC), "anthropic", "anthropic", int64(4), int64(0))
	mock.ExpectQuery(`(?s)WITH bucketed AS MATERIALIZED.*ORDER BY series_rank, dimension_key, display_bucket`).
		WithArgs(60, "platform", start, end, 60, 10).
		WillReturnRows(rows)
	mock.ExpectQuery(`SELECT MAX\(bucket_start\) FROM ops_rpm_metrics`).
		WithArgs(60, start, end).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(time.Date(2026, 8, 5, 8, 2, 0, 0, time.UTC)))

	response, err := repo.GetRPMTrend(context.Background(), filter)
	require.NoError(t, err)
	require.Len(t, response.Series, 2)
	require.Equal(t, "openai", response.Series[0].Key)
	require.Equal(t, "anthropic", response.Series[1].Key)
	require.Len(t, response.Series[0].Points, 3)
	require.Equal(t, float64(13), response.Series[0].Points[0].TotalRPM)
	require.Equal(t, float64(0), response.Series[0].Points[1].TotalRPM)
	require.Equal(t, float64(6), response.Series[0].Points[2].TotalRPM)
	require.NotNil(t, response.CompleteThrough)
	require.Equal(t, time.Date(2026, 8, 5, 8, 3, 0, 0, time.UTC), *response.CompleteThrough)
	require.NoError(t, mock.ExpectationsWereMet())
}

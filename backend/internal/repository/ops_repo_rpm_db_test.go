package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestRPMPostgresCoverageAndRecovery(t *testing.T) {
	dsn := os.Getenv("GROUP_HISTORY_TEST_DSN")
	if dsn == "" {
		t.Skip("set GROUP_HISTORY_TEST_DSN to an isolated local PostgreSQL database")
	}
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Contains(t, []string{"127.0.0.1", "localhost", "::1"}, parsed.Hostname())
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = admin.Close() }()
	schema := fmt.Sprintf("rpm_review_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	defer func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE") }()
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	_, err = db.Exec(`CREATE TABLE accounts(id bigint,name text,platform text);
 CREATE TABLE users(id bigint,username text,email text);
 CREATE TABLE usage_logs(created_at timestamptz,requested_model text,model text,account_id bigint,user_id bigint);
 CREATE TABLE ops_error_logs(created_at timestamptz,platform text,requested_model text,model text,account_id bigint,user_id bigint,status_code int,is_count_tokens boolean);
 INSERT INTO accounts VALUES(1,'A','openai'); INSERT INTO users VALUES(1,'U','');`)
	require.NoError(t, err)
	for _, name := range []string{"192_ops_rpm_minute_metrics.sql", "239_ops_rpm_coverage.sql"} {
		raw, err := migrations.FS.ReadFile(name)
		require.NoError(t, err)
		tx, err := db.Begin()
		require.NoError(t, err)
		_, err = tx.Exec(string(raw))
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
	}
	start := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO usage_logs VALUES($1,'gpt','gpt',1,1),($2,'gpt','gpt',1,1)`, start, start.Add(10*time.Minute))
	require.NoError(t, err)
	repo := &opsRepository{db: db}
	ctx := context.Background()
	// A coverage write failure must roll back the counts as well.
	_, err = db.Exec(`CREATE FUNCTION reject_coverage() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected'; END $$;
 CREATE TRIGGER reject_coverage BEFORE INSERT ON ops_rpm_coverage FOR EACH ROW EXECUTE FUNCTION reject_coverage()`)
	require.NoError(t, err)
	require.Error(t, repo.UpsertRPMMinuteMetrics(ctx, start, start.Add(time.Minute)))
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM ops_rpm_metrics").Scan(&count))
	require.Zero(t, count)
	_, err = db.Exec("DROP TRIGGER reject_coverage ON ops_rpm_coverage")
	require.NoError(t, err)
	require.NoError(t, repo.UpsertRPMMinuteMetrics(ctx, start, start.Add(2*time.Minute)))
	require.NoError(t, repo.UpsertRPMMinuteMetrics(ctx, start.Add(10*time.Minute), start.Add(11*time.Minute)))
	filter := &service.OpsRPMTrendFilter{StartTime: start, EndTime: start.Add(12 * time.Minute), Dimension: service.OpsRPMDimensionPlatform, SourceBucketSeconds: 60, OutputBucketSeconds: 60, TopN: 10}
	result, err := repo.GetRPMTrend(ctx, filter)
	require.NoError(t, err)
	require.True(t, result.Partial)
	points := result.Series[0].Points
	require.Equal(t, 1.0, *points[0].TotalRPM)
	require.Equal(t, 0.0, *points[1].TotalRPM) // collected idle minute
	require.Nil(t, points[2].TotalRPM)         // outage, not zero traffic
	require.Nil(t, points[11].TotalRPM)        // not yet collected
	require.NoError(t, repo.UpsertRPMRollup(ctx, 60, 300, start, start.Add(10*time.Minute)))
	filter.SourceBucketSeconds = 300
	filter.OutputBucketSeconds = 300
	filter.EndTime = start.Add(10 * time.Minute)
	result, err = repo.GetRPMTrend(ctx, filter)
	require.NoError(t, err)
	require.Nil(t, result.Series[0].Points[0].TotalRPM) // incomplete five-minute bucket
	// Catch-up and idempotent retries restore counts and genuine zeroes.
	require.NoError(t, repo.UpsertRPMMinuteMetrics(ctx, start, start.Add(12*time.Minute)))
	require.NoError(t, repo.UpsertRPMMinuteMetrics(ctx, start, start.Add(12*time.Minute)))
	require.NoError(t, repo.UpsertRPMRollup(ctx, 60, 300, start, start.Add(10*time.Minute)))
	result, err = repo.GetRPMTrend(ctx, filter)
	require.NoError(t, err)
	require.False(t, result.Partial)
	require.Equal(t, 0.2, *result.Series[0].Points[0].TotalRPM)
	require.Equal(t, 0.0, *result.Series[0].Points[1].TotalRPM)
	progress, err := repo.GetRPMCollectionEnd(ctx, 60)
	require.NoError(t, err)
	require.Equal(t, start.Add(12*time.Minute), *progress)
	// A partial requested display bucket must not be divided by a full duration.
	filter.StartTime = start.Add(30 * time.Second)
	filter.SourceBucketSeconds = 60
	filter.EndTime = start.Add(12 * time.Minute)
	result, err = repo.GetRPMTrend(ctx, filter)
	require.NoError(t, err)
	require.Nil(t, result.Series[0].Points[0].TotalRPM)
	require.NoError(t, repo.CleanupRPMMetrics(ctx, start.Add(12*time.Minute), start.Add(10*time.Minute), start))
	progress, err = repo.GetRPMCollectionEnd(ctx, 60)
	require.NoError(t, err)
	require.Nil(t, progress)
}

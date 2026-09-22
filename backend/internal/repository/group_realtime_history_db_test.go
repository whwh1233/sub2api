package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/alicebob/miniredis/v2"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGroupHistoryPostgresDurability(t *testing.T) {
	dsn := os.Getenv("GROUP_HISTORY_TEST_DSN")
	if dsn == "" {
		t.Skip("set GROUP_HISTORY_TEST_DSN to an isolated local PostgreSQL database")
	}
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Contains(t, []string{"127.0.0.1", "localhost", "::1"}, parsed.Hostname(), "local database only")
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer admin.Close()
	ctx := context.Background()
	schema := fmt.Sprintf("group_history_test_%d", time.Now().UnixNano())
	_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	defer admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	require.NoError(t, err)
	defer db.Close()
	_, err = db.ExecContext(ctx, `CREATE TABLE groups(id bigint PRIMARY KEY,name text,platform text,status text,sort_order int DEFAULT 0,deleted_at timestamptz); INSERT INTO groups(id,name,platform,status) VALUES(1,'Alpha','openai','active'),(2,'Beta','claude','active'),(3,'Idle','openai','active')`)
	require.NoError(t, err)
	migration, err := migrations.FS.ReadFile("238_ops_group_minute_metrics.sql")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, string(migration))
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
	}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	defer rdb.Close()
	store := NewGroupRealtimeStore(rdb, db).(*groupRealtimeCache)
	var now time.Time
	require.NoError(t, db.QueryRowContext(ctx, "SELECT NOW()").Scan(&now))
	end := now.UTC().Truncate(5 * time.Minute)
	start := end.Add(-15 * time.Minute)
	mr.Set(groupHistoryPrefix+"since", fmt.Sprint(start.Unix()))
	mr.SetTime(start)
	for _, metric := range []string{"started", "started", "success", "failed:timeout", "cancelled"} {
		require.NoError(t, store.Increment(ctx, 1, metric))
	}
	for _, metric := range []string{"started", "success"} {
		require.NoError(t, store.Increment(ctx, 2, metric))
	}
	mr.SetTime(end)
	// A failed batch must not leave a coverage marker or partially saved counts.
	_, err = db.ExecContext(ctx, `CREATE FUNCTION reject_history_group() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.group_id=2 THEN RAISE EXCEPTION 'injected write failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_history_group BEFORE INSERT ON ops_group_minute_metrics FOR EACH ROW EXECUTE FUNCTION reject_history_group()`)
	require.NoError(t, err)
	require.Error(t, store.PersistHistory(ctx))
	var afterFailure int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ops_group_minute_metrics").Scan(&afterFailure))
	require.Zero(t, afterFailure)
	_, err = db.ExecContext(ctx, "DROP TRIGGER reject_history_group ON ops_group_minute_metrics")
	require.NoError(t, err)
	// Flush, retry, and flush again: absolute upserts cannot double the counts.
	require.NoError(t, store.PersistHistory(ctx))
	require.NoError(t, store.PersistHistory(ctx))
	var count, started int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(started_count),0) FROM ops_group_minute_metrics`).Scan(&count, &started))
	require.EqualValues(t, 17, count)
	require.EqualValues(t, 3, started)
	// A second instance must respect the DB transaction lock.
	lock, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lock.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", groupHistoryPersistLock)
	require.NoError(t, err)
	require.NoError(t, store.PersistHistory(ctx))
	require.NoError(t, lock.Rollback())
	// Losing Redis cannot remove the saved history or its group curves.
	mr.FlushAll()
	mr.SetError("ERR unavailable")
	history, err := store.History(ctx, 60, nil)
	require.NoError(t, err)
	require.Equal(t, 15*60, history.CoveredSeconds)
	require.Len(t, history.Groups, 3)
	require.EqualValues(t, 1, history.Groups[0].GroupID)
	require.EqualValues(t, 2, history.Groups[0].Started)
	require.Equal(t, 50.0, *history.Groups[0].SuccessRate)
	for _, row := range history.Groups {
		require.Len(t, row.Points, len(history.Points))
	}
	for _, minutes := range []int{360, 1440} {
		h, err := store.History(ctx, minutes, nil)
		require.NoError(t, err)
		require.Equal(t, 15*60, h.CoveredSeconds)
		require.EqualValues(t, 2, h.Groups[0].Started)
	}
	week, err := store.History(ctx, 10080, nil)
	require.NoError(t, err)
	require.Equal(t, 900, week.BucketSeconds)
	require.Len(t, week.Points, 672)
	require.Equal(t, 7*24*time.Hour, week.EndTime.Sub(week.StartTime))
	for _, row := range week.Groups {
		require.Len(t, row.Points, 672)
	}
	_, err = store.History(ctx, 10081, nil)
	require.Error(t, err)
	mr.SetError("")
	// A new Redis epoch must not overwrite saved buckets with empty data.
	mr.SetTime(end.Add(30 * time.Second))
	require.NoError(t, store.PersistHistory(ctx))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT SUM(started_count) FROM ops_group_minute_metrics`).Scan(&started))
	require.EqualValues(t, 3, started)
	// A deleted group retains its captured name and counters.
	_, err = db.ExecContext(ctx, "UPDATE groups SET deleted_at=NOW() WHERE id=1")
	require.NoError(t, err)
	h, err := store.History(ctx, 60, nil)
	require.NoError(t, err)
	require.Equal(t, "Alpha", h.Groups[0].GroupName)
	// A late-reported outage invalidates an older complete bucket as well.
	require.NoError(t, rdb.ZAdd(ctx, groupHistoryPrefix+"gaps", redis.Z{Score: float64(start.Unix() + 20), Member: fmt.Sprint(start.Unix())}).Err())
	require.NoError(t, store.PersistHistory(ctx))
	h, err = store.History(ctx, 60, nil)
	require.NoError(t, err)
	require.Equal(t, 14*60, h.CoveredSeconds)
	for _, row := range h.Groups {
		require.Zero(t, row.Started)
	}
	// A missing minute invalidates its entire five-minute display bucket.
	_, err = db.ExecContext(ctx, `DELETE FROM ops_group_minute_metrics WHERE group_id=-1 AND bucket_start=$1`, start.Add(7*time.Minute))
	require.NoError(t, err)
	h, err = store.History(ctx, 360, nil)
	require.NoError(t, err)
	require.Equal(t, 5*60, h.CoveredSeconds)
	// Historical persistence has no automatic 24h/25h deletion.
	_, err = db.ExecContext(ctx, `INSERT INTO ops_group_minute_metrics(bucket_start,group_id,started_count) VALUES($1,1,9)`, end.Add(-40*24*time.Hour))
	require.NoError(t, err)
	require.NoError(t, store.PersistHistory(ctx))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT started_count FROM ops_group_minute_metrics WHERE bucket_start=$1 AND group_id=1`, end.Add(-40*24*time.Hour)).Scan(&started))
	require.EqualValues(t, 9, started)
	// Weekly queries include persisted data beyond Redis retention.
	oldBucket := end.Add(-6 * 24 * time.Hour).Truncate(15 * time.Minute)
	_, err = db.ExecContext(ctx, `INSERT INTO ops_group_minute_metrics(bucket_start,group_id)
SELECT generate_series($1::timestamptz,$1::timestamptz+interval '14 minutes',interval '1 minute'),-1`, oldBucket)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO ops_group_minute_metrics(bucket_start,group_id,started_count,success_count) VALUES($1,2,150,120)`, oldBucket)
	require.NoError(t, err)
	week, err = store.History(ctx, 10080, nil)
	require.NoError(t, err)
	found := false
	for _, point := range week.Points {
		if point.BucketStart.Equal(oldBucket) {
			found = true
			require.False(t, point.Partial)
			require.EqualValues(t, 150, point.Started)
			require.Equal(t, 10.0, *point.RPM)
		}
	}
	require.True(t, found)

}

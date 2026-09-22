package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// The fixture deliberately has no Redis client/server: both APIs and the
// historical collector must work with PostgreSQL alone.
func TestGroupRealtimeDatabaseOnly(t *testing.T) {
	dsn := os.Getenv("GROUP_HISTORY_TEST_DSN")
	if dsn == "" {
		t.Skip("set local GROUP_HISTORY_TEST_DSN")
	}
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Contains(t, []string{"127.0.0.1", "localhost", "::1"}, parsed.Hostname())
	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = admin.Close() }()
	schema := fmt.Sprintf("group_logs_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	defer func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE") }()
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	db, err := sql.Open("postgres", parsed.String())
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(8)
	_, err = db.Exec(`CREATE TABLE groups(id bigint PRIMARY KEY,name text,platform text,status text,sort_order int DEFAULT 0,deleted_at timestamptz);
 INSERT INTO groups VALUES(1,'Active','openai','active',0,NULL),(2,'Idle','openai','active',1,NULL),(3,'Deleted','openai','active',2,NOW());
 CREATE TABLE usage_logs(created_at timestamptz,group_id bigint);
 CREATE TABLE ops_error_logs(created_at timestamptz,group_id bigint,status_code int,is_count_tokens boolean);`)
	require.NoError(t, err)
	for _, name := range []string{"238_ops_group_minute_metrics.sql", "240_ops_group_log_minute_metrics.sql"} {
		raw, e := migrations.FS.ReadFile(name)
		require.NoError(t, e)
		// New migration is repeatable and never modifies already-saved ingress history.
		for i := 0; i < 2; i++ {
			tx, e := db.Begin()
			require.NoError(t, e)
			_, e = tx.Exec(string(raw))
			require.NoError(t, e)
			require.NoError(t, tx.Commit())
		}
	}
	_, err = db.Exec(`INSERT INTO ops_group_minute_metrics(bucket_start,group_id,started_count) VALUES(date_trunc('minute',NOW())-interval '1 day',1,999)`)
	require.NoError(t, err)
	var now time.Time
	require.NoError(t, db.QueryRow("SELECT NOW()").Scan(&now))
	recent := now.Add(-5 * time.Second)
	old := now.Add(-2 * time.Minute)
	_, err = db.Exec(`INSERT INTO usage_logs VALUES($1,1),($1,1),($1,3),($2,1);
 `, recent, old)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ops_error_logs VALUES($1,1,503,false),($1,1,200,false),($1,1,400,true),($1,1,499,false),($1,NULL,401,false),($2,1,500,false)`, recent, old)
	require.NoError(t, err)
	store := NewGroupRealtimeStore(db).(*groupRealtimeStore)
	ctx := context.Background()
	live, err := store.Snapshot(ctx)
	require.NoError(t, err)
	require.Len(t, live.Groups, 4)
	require.EqualValues(t, 1, live.Groups[0].GroupID)
	require.EqualValues(t, 4, live.Groups[0].RPM)
	require.EqualValues(t, 2, live.Groups[0].Success)
	require.EqualValues(t, 1, live.Groups[0].Failed)
	require.EqualValues(t, 1, live.Groups[0].Cancelled)
	require.InDelta(t, 100.0*2/3, *live.Groups[0].SuccessRate, 0.001)
	require.EqualValues(t, 1, live.Groups[0].Failures["logged_error"])
	for _, row := range live.Groups {
		if row.GroupID == 2 {
			require.Zero(t, row.RPM)
			require.Nil(t, row.SuccessRate)
		}
	}
	before, err := store.History(ctx, 15, nil)
	require.NoError(t, err)
	require.Zero(t, before.CoveredSeconds)
	require.NoError(t, store.PersistHistory(ctx))
	require.NoError(t, store.PersistHistory(ctx))
	after, err := store.History(ctx, 15, nil)
	require.NoError(t, err)
	require.Equal(t, 5*60, after.CoveredSeconds)
	require.EqualValues(t, 2, after.Groups[0].Started)
	require.EqualValues(t, 1, after.Groups[0].Success)
	require.EqualValues(t, 1, after.Groups[0].Failed)
	require.Equal(t, 50.0, *after.Groups[0].SuccessRate)
	// Late log inside overlap: replace totals, never increment an existing rollup.
	_, err = db.Exec("INSERT INTO usage_logs VALUES($1,1)", old)
	require.NoError(t, err)
	require.NoError(t, store.PersistHistory(ctx))
	after, err = store.History(ctx, 15, nil)
	require.NoError(t, err)
	require.EqualValues(t, 3, after.Groups[0].Started)
	// Failed insert rolls back deletion and coverage as well as group counters.
	_, err = db.Exec(`CREATE FUNCTION reject_group_rollup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.group_id=1 THEN RAISE EXCEPTION 'injected failure'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER reject_group_rollup BEFORE INSERT ON ops_group_log_minute_metrics FOR EACH ROW EXECUTE FUNCTION reject_group_rollup()`)
	require.NoError(t, err)
	require.Error(t, store.PersistHistory(ctx))
	still, err := store.History(ctx, 15, nil)
	require.NoError(t, err)
	require.EqualValues(t, 3, still.Groups[0].Started)
	require.Equal(t, after.CoveredSeconds, still.CoveredSeconds)
	_, err = db.Exec("DROP TRIGGER reject_group_rollup ON ops_group_log_minute_metrics")
	require.NoError(t, err)
	var wg sync.WaitGroup
	errs := make([]error, 12)
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); errs[i] = store.PersistHistory(ctx) }(i)
	}
	wg.Wait()
	for _, e := range errs {
		require.NoError(t, e)
	}
	after, err = store.History(ctx, 15, nil)
	require.NoError(t, err)
	require.EqualValues(t, 3, after.Groups[0].Started)
	// Deleted group metadata survives; unscanned history is a gap, not zero.
	week, err := store.History(ctx, 10080, nil)
	require.NoError(t, err)
	require.Len(t, week.Points, 672)
	require.True(t, week.Partial)
	var legacy int
	require.NoError(t, db.QueryRow("SELECT SUM(started_count) FROM ops_group_minute_metrics").Scan(&legacy))
	require.Equal(t, 999, legacy)
	// Persisted SQL progress catches up an outage in at most 15-minute batches.
	_, err = db.Exec("DELETE FROM ops_group_log_minute_metrics")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ops_group_log_minute_metrics(bucket_start,group_id) VALUES(date_trunc('minute',NOW())-interval '21 minutes',-1)`)
	require.NoError(t, err)
	require.NoError(t, store.PersistHistory(ctx))
	var lag float64
	require.NoError(t, db.QueryRow("SELECT EXTRACT(EPOCH FROM date_trunc('minute',NOW())-MAX(bucket_start)) FROM ops_group_log_minute_metrics WHERE group_id=-1").Scan(&lag))
	require.Equal(t, 660.0, lag)
	require.NoError(t, store.PersistHistory(ctx))
	require.NoError(t, db.QueryRow("SELECT EXTRACT(EPOCH FROM date_trunc('minute',NOW())-MAX(bucket_start)) FROM ops_group_log_minute_metrics WHERE group_id=-1").Scan(&lag))
	require.Equal(t, 60.0, lag)
}

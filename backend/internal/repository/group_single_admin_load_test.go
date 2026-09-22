package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// Run only against a fresh, disposable local database. No inference or Redis.
func TestGroupSingleAdminLoad(t *testing.T) {
	if os.Getenv("SINGLE_ADMIN_LOAD") != "1" {
		t.Skip("isolated opt-in load test")
	}
	db, err := sql.Open("postgres", "postgres://postgres@127.0.0.1:55440/single_admin?sslmode=disable")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(16)
	_, err = db.Exec(`CREATE TABLE groups(id bigint PRIMARY KEY,name text,platform text,status text,sort_order int DEFAULT 0,deleted_at timestamptz);
 CREATE TABLE accounts(id bigint PRIMARY KEY,name text,platform text);
 CREATE TABLE users(id bigint PRIMARY KEY,username text,email text);
 CREATE TABLE usage_logs(created_at timestamptz,group_id bigint,requested_model text,model text,account_id bigint,user_id bigint);
 CREATE INDEX ON usage_logs(created_at);
 CREATE TABLE ops_error_logs(created_at timestamptz,group_id bigint,platform text,requested_model text,model text,account_id bigint,user_id bigint,status_code int,is_count_tokens boolean);
 CREATE INDEX ON ops_error_logs(created_at);
 INSERT INTO groups SELECT i,'Group '||i,'openai','active',0,NULL FROM generate_series(1,10) i;
 INSERT INTO accounts SELECT i,'Account '||i,'openai' FROM generate_series(1,551) i;
 INSERT INTO users SELECT i,'User '||i,'' FROM generate_series(1,1012) i;`)
	require.NoError(t, err)
	for _, name := range []string{"192_ops_rpm_minute_metrics.sql", "238_ops_group_minute_metrics.sql", "239_ops_rpm_coverage.sql", "240_ops_group_log_minute_metrics.sql"} {
		raw, e := migrations.FS.ReadFile(name)
		require.NoError(t, e)
		tx, e := db.Begin()
		require.NoError(t, e)
		_, e = tx.Exec(string(raw))
		require.NoError(t, e)
		require.NoError(t, tx.Commit())
	}
	var end time.Time
	require.NoError(t, db.QueryRow("SELECT date_trunc('minute',NOW())").Scan(&end))
	start := end.Add(-7 * 24 * time.Hour)
	_, err = db.Exec(`INSERT INTO usage_logs SELECT $1::timestamptz+i*interval '1.5 seconds',i%10+1,'gpt','gpt',i%3+1,i%6+1 FROM generate_series(0,403199) i`, start)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ops_group_log_minute_metrics(bucket_start,group_id,group_name,platform,status,started_count,success_count)
 SELECT b,i,'Group '||i,'openai','active',4,4 FROM generate_series($1::timestamptz,$2::timestamptz-interval '1 minute',interval '1 minute') b CROSS JOIN generate_series(1,10) i`, start, end)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ops_group_log_minute_metrics(bucket_start,group_id) SELECT b,-1 FROM generate_series($1::timestamptz,$2::timestamptz-interval '1 minute',interval '1 minute') b`, start, end)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ops_rpm_metrics(bucket_start,bucket_seconds,dimension_type,dimension_key,dimension_label,success_count,error_count)
 SELECT b,60,'account',i::text,'Account '||i,CASE WHEN i=1 THEN 14 ELSE 13 END,0 FROM generate_series($1::timestamptz,$2::timestamptz-interval '1 minute',interval '1 minute') b CROSS JOIN generate_series(1,3) i`, start, end)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ops_rpm_coverage SELECT 60,b FROM generate_series($1::timestamptz,$2::timestamptz-interval '1 minute',interval '1 minute') b`, start, end)
	require.NoError(t, err)
	_, err = db.Exec("ANALYZE")
	require.NoError(t, err)
	t.Log("SEED groups=10 accounts=551 users=1012 usage_rows=403200 group_history_rows=110880 rpm_rows=30240 seven_days average=40RPM")
	store := NewGroupRealtimeStore(db).(*groupRealtimeStore)
	repo := &opsRepository{db: db}
	for _, phase := range []struct {
		name     string
		duration time.Duration
		page     bool
	}{{"baseline", 30 * time.Second, false}, {"single_admin", 125 * time.Second, true}} {
		ctx, cancel := context.WithTimeout(context.Background(), phase.duration)
		var wg sync.WaitGroup
		var mu sync.Mutex
		timings := map[string][]time.Duration{}
		errors := map[string]int{}
		record := func(name string, fn func(context.Context) error) {
			begin := time.Now()
			err := fn(ctx)
			elapsed := time.Since(begin)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if ctx.Err() == nil {
					errors[name]++
					t.Logf("ERROR %s %v", name, err)
				}
				return
			}
			timings[name] = append(timings[name], elapsed)
		}
		launch := func(name string, interval time.Duration, fn func(context.Context) error) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				timer := time.NewTicker(interval)
				defer timer.Stop()
				record(name, fn)
				for {
					select {
					case <-ctx.Done():
						return
					case <-timer.C:
						record(name, fn)
					}
				}
			}()
		}
		launch("business_lookup", 50*time.Millisecond, func(c context.Context) error {
			var name string
			return db.QueryRowContext(c, "SELECT name FROM accounts WHERE id=1").Scan(&name)
		})
		launch("usage_write", 1500*time.Millisecond, func(c context.Context) error {
			_, e := db.ExecContext(c, "INSERT INTO usage_logs VALUES(NOW(),1,'gpt','gpt',1,1)")
			return e
		})
		if phase.page {
			launch("live_5s", 5*time.Second, func(c context.Context) error {
				c, stop := context.WithTimeout(c, 3*time.Second)
				defer stop()
				v, e := store.Snapshot(c)
				if e != nil {
					return e
				}
				if len(v.Groups) != 10 {
					return fmt.Errorf("unexpected groups")
				}
				_, e = json.Marshal(v)
				return e
			})
			launch("history_30s", 30*time.Second, func(c context.Context) error {
				c, stop := context.WithTimeout(c, 5*time.Second)
				defer stop()
				v, e := store.History(c, 10080, nil)
				if e != nil {
					return e
				}
				if len(v.Points) != 672 || len(v.Groups) != 10 {
					return fmt.Errorf("unexpected week dimensions")
				}
				_, e = json.Marshal(v)
				return e
			})
			// One extra Ops trend tab: slightly heavier than the group page alone.
			launch("ops_rpm_30s", 30*time.Second, func(c context.Context) error {
				v, e := repo.GetRPMTrend(c, &service.OpsRPMTrendFilter{StartTime: start, EndTime: end, Dimension: service.OpsRPMDimensionAccount, SourceBucketSeconds: 60, OutputBucketSeconds: 1800, TopN: 10})
				if e != nil {
					return e
				}
				_, e = json.Marshal(v)
				return e
			})
		}
		// Both phases include the same production-style background aggregation.
		launch("collector_60s", 60*time.Second, func(c context.Context) error {
			c, stop := context.WithTimeout(c, 20*time.Second)
			defer stop()
			if e := store.PersistHistory(c); e != nil {
				return e
			}
			now := time.Now().UTC().Truncate(time.Minute)
			if e := repo.UpsertRPMMinuteMetrics(c, now.Add(-5*time.Minute), now); e != nil {
				return e
			}
			five := now.Truncate(5 * time.Minute)
			return repo.UpsertRPMRollup(c, 60, 300, five.Add(-5*time.Minute), five)
		})
		<-ctx.Done()
		wg.Wait()
		cancel()
		keys := make([]string, 0, len(timings))
		for k := range timings {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := timings[k]
			sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
			t.Logf("RESULT phase=%s operation=%s n=%d errors=%d p50=%s p95=%s max=%s", phase.name, k, len(v), errors[k], v[len(v)/2], v[(len(v)-1)*95/100], v[len(v)-1])
		}
		for k, v := range errors {
			require.Zerof(t, v, "%s %s", phase.name, k)
		}
		require.Eventually(t, func() bool { return db.Stats().InUse == 0 }, time.Second, 10*time.Millisecond)
		t.Logf("PHASE %s complete inuse=%d waits=%d", phase.name, db.Stats().InUse, db.Stats().WaitCount)
	}
}

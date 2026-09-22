package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// PostgreSQL is the only source. This store has no request-path write API and
// does not depend on Redis. The definition matches log-based Ops reporting:
// usage rows, final error rows (not recovered 2xx attempts or count_tokens),
// and explicit 499 cancellations. Logging filters/late writes affect counts.
type groupRealtimeStore struct{ db *sql.DB }

func NewGroupRealtimeStore(db *sql.DB) service.GroupRealtimeStore {
	return &groupRealtimeStore{db: db}
}

// Both live and historical reports MUST use this same bounded source query.
// Count log records, not upstream attempts. Do not deduplicate by request_id:
// usage and error logs can use different upstream/local IDs and WS turns may
// legitimately share an ID. Incomplete billed streams may also have error rows.
const groupLogEventsSQL = `WITH events AS (
 SELECT created_at,COALESCE(group_id,0) AS group_id,'success'::text AS outcome
 FROM usage_logs WHERE created_at >= $1 AND created_at < $2
 UNION ALL
 SELECT created_at,COALESCE(group_id,0),CASE WHEN status_code=499 THEN 'cancelled' ELSE 'failed' END
 FROM ops_error_logs WHERE created_at >= $1 AND created_at < $2
 AND status_code >= 400 AND is_count_tokens=FALSE
) `

func (r *groupRealtimeStore) Snapshot(ctx context.Context) (*service.GroupRealtimeSnapshot, error) {
	var end time.Time
	if err := r.db.QueryRowContext(ctx, "SELECT date_trunc('second',NOW())").Scan(&end); err != nil {
		return nil, err
	}
	end = end.UTC()
	start := end.Add(-time.Minute)
	rows, err := r.db.QueryContext(ctx, groupLogEventsSQL+`, counts AS (
 SELECT group_id,COUNT(*) AS total,COUNT(*) FILTER(WHERE outcome='success') AS success,
 COUNT(*) FILTER(WHERE outcome='failed') AS failed,COUNT(*) FILTER(WHERE outcome='cancelled') AS cancelled
 FROM events GROUP BY group_id
 ) SELECT COALESCE(g.id,c.group_id),COALESCE(g.name,''),COALESCE(g.platform,''),COALESCE(g.status,'unknown'),
 COALESCE(c.total,0),COALESCE(c.success,0),COALESCE(c.failed,0),COALESCE(c.cancelled,0)
 FROM groups g FULL JOIN counts c ON g.id=c.group_id
 WHERE (g.id IS NOT NULL AND g.deleted_at IS NULL) OR c.group_id IS NOT NULL
 ORDER BY COALESCE(c.total,0) DESC,COALESCE(g.id,c.group_id)`, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := &service.GroupRealtimeSnapshot{StartTime: start, EndTime: end, CollectingSince: start, Groups: []service.GroupRealtimeRow{}}
	for rows.Next() {
		row := service.GroupRealtimeRow{Failures: map[string]int64{}}
		if err := rows.Scan(&row.GroupID, &row.GroupName, &row.Platform, &row.Status, &row.RPM, &row.Success, &row.Failed, &row.Cancelled); err != nil {
			return nil, err
		}
		row.SuccessRate = groupHistoryRate(row.Success, row.Failed)
		// Detailed reasons remain available in Ops error logs. Do not fabricate
		// request-path classifications from persisted, possibly filtered records.
		if row.Failed > 0 {
			row.Failures["logged_error"] = row.Failed
		}
		result.Groups = append(result.Groups, row)
	}
	return result, rows.Err()
}

// PersistHistory runs only in the background collector. Recompute a five-minute
// overlap for delayed log writes and resume progress in bounded batches.
func (r *groupRealtimeStore) PersistHistory(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Bound SQL work on a responsive DB; these do not replace network deadlines.
	if _, err = tx.ExecContext(ctx, "SET LOCAL statement_timeout='8s'; SET LOCAL lock_timeout='1s'"); err != nil {
		return err
	}
	var locked bool
	if err = tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", int64(738241902)).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return nil
	}
	var end time.Time
	var latest sql.NullTime
	if err = tx.QueryRowContext(ctx, `SELECT date_trunc('minute',NOW()),MAX(bucket_start) FROM ops_group_log_minute_metrics WHERE group_id=-1`).Scan(&end, &latest); err != nil {
		return err
	}
	end = end.UTC()
	start := end.Add(-5 * time.Minute)
	if latest.Valid && latest.Time.Add(-4*time.Minute).Before(start) {
		start = latest.Time.UTC().Add(-4 * time.Minute)
	}
	if earliest := end.Add(-24 * time.Hour); start.Before(earliest) {
		start = earliest
	}
	if batchEnd := start.Add(15 * time.Minute); batchEnd.Before(end) {
		end = batchEnd
	}
	// A single statement snapshots logs consistently. Replace absolute values so
	// retries do not double counts, and removals in the overlap do not leave stale rows.
	if _, err = tx.ExecContext(ctx, `DELETE FROM ops_group_log_minute_metrics WHERE bucket_start >= $1 AND bucket_start < $2`, start, end); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, groupLogEventsSQL+`, counts AS (
 SELECT date_trunc('minute',created_at) AS bucket_start,group_id,COUNT(*) AS total,
 COUNT(*) FILTER(WHERE outcome='success') AS success,COUNT(*) FILTER(WHERE outcome='failed') AS failed,
 COUNT(*) FILTER(WHERE outcome='cancelled') AS cancelled FROM events GROUP BY 1,2
 ) INSERT INTO ops_group_log_minute_metrics(bucket_start,group_id,group_name,platform,status,started_count,success_count,failed_count,cancelled_count)
 SELECT c.bucket_start,c.group_id,COALESCE(g.name,''),COALESCE(g.platform,''),COALESCE(g.status,'unknown'),c.total,c.success,c.failed,c.cancelled
 FROM counts c LEFT JOIN groups g ON g.id=c.group_id
 UNION ALL
 SELECT minute,-1,'','','',0,0,0,0 FROM generate_series($1::timestamptz,$2::timestamptz-interval '1 minute',interval '1 minute') minute`, start, end)
	if err != nil {
		return err
	}
	return tx.Commit()
}

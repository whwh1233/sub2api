package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// Coverage commits atomically with group counts. -1 is metadata; 0 remains the
// unassigned group. Idle groups need no individual rows for each quiet minute.
const groupHistoryCoverageID int64 = -1
const groupHistoryPersistLock int64 = 738241901

type groupHistoryDBRow struct {
	BucketStart time.Time `json:"bucket_start"`
	GroupID     int64     `json:"group_id"`
	GroupName   string    `json:"group_name"`
	Platform    string    `json:"platform"`
	Status      string    `json:"status"`
	Started     int64     `json:"started_count"`
	Success     int64     `json:"success_count"`
	Failed      int64     `json:"failed_count"`
	Cancelled   int64     `json:"cancelled_count"`
	Partial     bool      `json:"partial"`
}

func (r *groupRealtimeCache) PersistHistory(ctx context.Context) error {
	if err := r.publishGap(ctx); err != nil {
		return err
	}
	now, err := r.rdb.Time(ctx).Result()
	if err != nil {
		return err
	}
	if err := r.rdb.SetNX(ctx, groupHistoryPrefix+"since", now.Unix(), 0).Err(); err != nil {
		return err
	}
	since, err := r.rdb.Get(ctx, groupHistoryPrefix+"since").Int64()
	if err != nil {
		return err
	}
	end := now.UTC().Truncate(time.Minute)
	// Never overwrite durable buckets predating the current Redis collection epoch.
	start := time.Unix(since, 0).UTC().Truncate(time.Minute)
	if cutoff := end.Add(-24 * time.Hour); start.Before(cutoff) {
		start = cutoff
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var locked bool
	if err := tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", groupHistoryPersistLock).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return nil
	}
	var latest sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT MAX(bucket_start) FROM ops_group_minute_metrics WHERE group_id = -1`).Scan(&latest); err != nil {
		return err
	}
	if latest.Valid && latest.Time.Add(-5*time.Minute).After(start) {
		start = latest.Time.Add(-5 * time.Minute)
	}
	if start.Before(end) {
		snapshot, err := r.readRedisHistoryWindow(ctx, start, end, 60, nil)
		if err != nil {
			return err
		}
		// A Redis reset during the read must never overwrite durable rows with
		// incomplete data from a different collection epoch.
		currentSince, err := r.rdb.Get(ctx, groupHistoryPrefix+"since").Int64()
		if err != nil {
			return err
		}
		if currentSince != since || snapshot.CollectingSince.Unix() != since {
			return fmt.Errorf("group history collection epoch changed during persistence")
		}
		payload := make([]groupHistoryDBRow, 0, len(snapshot.Points))
		for _, point := range snapshot.Points {
			payload = append(payload, groupHistoryDBRow{BucketStart: point.BucketStart, GroupID: groupHistoryCoverageID, Partial: point.Partial})
		}
		for _, group := range snapshot.Groups {
			for _, point := range group.Points {
				if point.Started+point.Success+point.Failed+point.Cancelled == 0 {
					continue
				}
				payload = append(payload, groupHistoryDBRow{BucketStart: point.BucketStart, GroupID: group.GroupID, GroupName: group.GroupName, Platform: group.Platform, Status: group.Status, Started: point.Started, Success: point.Success, Failed: point.Failed, Cancelled: point.Cancelled, Partial: point.Partial})
			}
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		// Absolute counts make retries and multi-instance writes idempotent.
		_, err = tx.ExecContext(ctx, `INSERT INTO ops_group_minute_metrics
   (bucket_start,group_id,group_name,platform,status,started_count,success_count,failed_count,cancelled_count,partial)
   SELECT bucket_start,group_id,group_name,platform,status,started_count,success_count,failed_count,cancelled_count,partial
   FROM jsonb_to_recordset($1::jsonb) AS x(bucket_start timestamptz,group_id bigint,group_name text,platform text,status text,started_count bigint,success_count bigint,failed_count bigint,cancelled_count bigint,partial boolean)
   ON CONFLICT(bucket_start,group_id) DO UPDATE SET
   group_name=EXCLUDED.group_name,platform=EXCLUDED.platform,status=EXCLUDED.status,
   started_count=EXCLUDED.started_count,success_count=EXCLUDED.success_count,
   failed_count=EXCLUDED.failed_count,cancelled_count=EXCLUDED.cancelled_count,
   partial=EXCLUDED.partial,updated_at=NOW()`, string(raw))
		if err != nil {
			return err
		}
	}
	// Another instance may report an older outage after recovery. Correct saved
	// coverage even when the gap predates the normal five-minute overlap.
	gaps, err := r.rdb.ZRangeByScoreWithScores(ctx, groupHistoryPrefix+"gaps", &redis.ZRangeBy{Min: strconv.FormatInt(end.Add(-25*time.Hour).Unix(), 10), Max: "+inf"}).Result()
	if err != nil {
		return err
	}
	for _, gap := range gaps {
		first, err := strconv.ParseInt(fmt.Sprint(gap.Member), 10, 64)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE ops_group_minute_metrics SET partial=TRUE,updated_at=NOW()
   WHERE bucket_start >= $1 AND bucket_start <= $2 AND partial=FALSE`, time.Unix(first, 0).UTC().Truncate(time.Minute), time.Unix(int64(gap.Score), 0).UTC().Truncate(time.Minute))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// History needs PostgreSQL only: Redis loss cannot erase saved history.
func (r *groupRealtimeCache) History(ctx context.Context, minutes int, groupID *int64) (*service.GroupRealtimeHistory, error) {
	switch minutes {
	case 15, 60, 360, 1440, 10080:
	default:
		return nil, fmt.Errorf("unsupported history window")
	}
	step := 60
	if minutes > 60 {
		step = 300
	}
	if minutes == 10080 {
		step = 900
	}
	var now time.Time
	var first sql.NullTime
	if err := r.db.QueryRowContext(ctx, `SELECT NOW(), MIN(bucket_start) FROM ops_group_minute_metrics WHERE group_id = -1`).Scan(&now, &first); err != nil {
		return nil, err
	}
	end := now.UTC().Truncate(time.Duration(step) * time.Second)
	start := end.Add(-time.Duration(minutes) * time.Minute)
	count := minutes * 60 / step
	result := &service.GroupRealtimeHistory{StartTime: start, EndTime: end, BucketSeconds: step, GroupID: groupID, Groups: []service.GroupRealtimeHistoryRow{}, Points: make([]service.GroupRealtimeHistoryPoint, count)}
	if first.Valid {
		result.CollectingSince = first.Time.UTC()
	}
	byID := map[int64]*service.GroupRealtimeHistoryRow{}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, platform, status FROM groups WHERE deleted_at IS NULL ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		row := &service.GroupRealtimeHistoryRow{Points: make([]service.GroupRealtimeHistoryPoint, count)}
		if err := rows.Scan(&row.GroupID, &row.GroupName, &row.Platform, &row.Status); err != nil {
			rows.Close()
			return nil, err
		}
		byID[row.GroupID] = row
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	// Coverage and counts share one MVCC statement, so a partial flush can never
	// expose a complete marker without the corresponding group data.
	rows, err = r.db.QueryContext(ctx, `SELECT date_bin(make_interval(secs => $3),bucket_start,TIMESTAMPTZ '1970-01-01 00:00:00+00') AS bucket,
 group_id,(array_agg(group_name ORDER BY bucket_start DESC))[1],
 (array_agg(platform ORDER BY bucket_start DESC))[1],(array_agg(status ORDER BY bucket_start DESC))[1],
 SUM(started_count),SUM(success_count),SUM(failed_count),SUM(cancelled_count),COUNT(*) FILTER (WHERE NOT partial)
 FROM ops_group_minute_metrics WHERE bucket_start >= $1 AND bucket_start < $2
 GROUP BY bucket,group_id ORDER BY bucket,group_id`, start, end, step)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	covered := make([]bool, count)
	for rows.Next() {
		var record groupHistoryDBRow
		var complete int
		if err := rows.Scan(&record.BucketStart, &record.GroupID, &record.GroupName, &record.Platform, &record.Status, &record.Started, &record.Success, &record.Failed, &record.Cancelled, &complete); err != nil {
			return nil, err
		}
		i := int(record.BucketStart.Sub(start).Seconds()) / step
		if i < 0 || i >= count {
			return nil, fmt.Errorf("history bucket outside query window")
		}
		if record.GroupID == groupHistoryCoverageID {
			covered[i] = complete == step/60
			continue
		}
		row := byID[record.GroupID]
		if row == nil {
			row = &service.GroupRealtimeHistoryRow{GroupID: record.GroupID, GroupName: record.GroupName, Platform: record.Platform, Status: record.Status, Points: make([]service.GroupRealtimeHistoryPoint, count)}
			byID[row.GroupID] = row
		}
		row.Points[i] = service.GroupRealtimeHistoryPoint{Started: record.Started, Success: record.Success, Failed: record.Failed, Cancelled: record.Cancelled}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range result.Points {
		point := &result.Points[i]
		point.BucketStart = start.Add(time.Duration(i*step) * time.Second)
		point.Partial = !covered[i]
		if point.Partial {
			result.Partial = true
		} else {
			result.CoveredSeconds += step
		}
	}
	for _, row := range byID {
		for i := range row.Points {
			point := &row.Points[i]
			point.BucketStart = result.Points[i].BucketStart
			point.Partial = !covered[i]
			if point.Partial {
				*point = service.GroupRealtimeHistoryPoint{BucketStart: point.BucketStart, Partial: true}
				continue
			}
			rpm := float64(point.Started) * 60 / float64(step)
			point.RPM = &rpm
			point.SuccessRate = groupHistoryRate(point.Success, point.Failed)
			row.Started += point.Started
			row.Success += point.Success
			row.Failed += point.Failed
			row.Cancelled += point.Cancelled
			if groupID == nil || *groupID == row.GroupID {
				total := &result.Points[i]
				total.Started += point.Started
				total.Success += point.Success
				total.Failed += point.Failed
				total.Cancelled += point.Cancelled
			}
		}
		if result.CoveredSeconds > 0 {
			rpm := float64(row.Started) * 60 / float64(result.CoveredSeconds)
			row.RPM = &rpm
		}
		row.SuccessRate = groupHistoryRate(row.Success, row.Failed)
		result.Groups = append(result.Groups, *row)
	}
	for i := range result.Points {
		point := &result.Points[i]
		if !point.Partial {
			rpm := float64(point.Started) * 60 / float64(step)
			point.RPM = &rpm
			point.SuccessRate = groupHistoryRate(point.Success, point.Failed)
		}
	}
	sort.Slice(result.Groups, func(i, j int) bool {
		if result.Groups[i].Started != result.Groups[j].Started {
			return result.Groups[i].Started > result.Groups[j].Started
		}
		return result.Groups[i].GroupID < result.Groups[j].GroupID
	})
	return result, nil
}

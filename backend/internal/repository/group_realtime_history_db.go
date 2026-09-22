package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"sort"
	"time"
)

const groupHistoryCoverageID int64 = -1

type groupHistoryDBRow struct {
	BucketStart                         time.Time
	GroupID                             int64
	GroupName, Platform, Status         string
	Started, Success, Failed, Cancelled int64
}

func groupHistoryRate(success, failed int64) *float64 {
	if success+failed == 0 {
		return nil
	}
	rate := 100 * float64(success) / float64(success+failed)
	return &rate
}

// History needs PostgreSQL only: Redis loss cannot erase saved history.
func (r *groupRealtimeStore) History(ctx context.Context, minutes int, groupID *int64) (*service.GroupRealtimeHistory, error) {
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
	if err := r.db.QueryRowContext(ctx, `SELECT NOW(), MIN(bucket_start) FROM ops_group_log_minute_metrics WHERE group_id = -1`).Scan(&now, &first); err != nil {
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
 FROM ops_group_log_minute_metrics WHERE bucket_start >= $1 AND bucket_start < $2
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

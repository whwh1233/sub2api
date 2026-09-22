package repository

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const groupHistoryPrefix = "ops:group-history:v1:"

// Estimate the unavailable interval using elapsed local time and the shared
// Redis clock. Retain its end as the score so overlapping queries can find it.
const groupHistoryGapLua = `
local now = tonumber(redis.call('TIME')[1])
redis.call('SET', ARGV[1] .. 'gap', '1', 'EX', 60)
redis.call('ZADD', ARGV[2] .. 'gaps', now, tostring(now - tonumber(ARGV[3])))
redis.call('ZREMRANGEBYSCORE', ARGV[2] .. 'gaps', '-inf', now - 90000)
redis.call('EXPIRE', ARGV[2] .. 'gaps', 90000)
return 1
`

func (r *groupRealtimeCache) readRedisHistory(ctx context.Context, minutes int, groupID *int64) (*service.GroupRealtimeHistory, error) {
	// Bound the number of Redis commands even for internal callers.
	switch minutes {
	case 15, 60, 360, 1440:
	default:
		return nil, fmt.Errorf("unsupported history window")
	}
	if err := r.publishGap(ctx); err != nil {
		return nil, err
	}
	now, err := r.rdb.Time(ctx).Result()
	if err != nil {
		return nil, err
	}
	step := 60
	if minutes > 60 {
		step = 300
	}
	end := now.UTC().Truncate(time.Duration(step) * time.Second)
	start := end.Add(-time.Duration(minutes) * time.Minute)
	return r.readRedisHistoryWindow(ctx, start, end, step, groupID)
}

func (r *groupRealtimeCache) readRedisHistoryWindow(ctx context.Context, start, end time.Time, step int, groupID *int64) (*service.GroupRealtimeHistory, error) {
	now, err := r.rdb.Time(ctx).Result()
	if err != nil {
		return nil, err
	}
	if err := r.rdb.SetNX(ctx, groupHistoryPrefix+"since", now.Unix(), 0).Err(); err != nil {
		return nil, err
	}
	pipe := r.rdb.Pipeline()
	sinceCmd := pipe.Get(ctx, groupHistoryPrefix+"since")
	gapCmd := pipe.ZRangeByScoreWithScores(ctx, groupHistoryPrefix+"gaps", &redis.ZRangeBy{Min: strconv.FormatInt(start.Unix(), 10), Max: "+inf"})
	commands := make([]*redis.MapStringStringCmd, 0, int(end.Sub(start).Seconds())/step)
	for bucket := start.Unix(); bucket < end.Unix(); bucket += int64(step) {
		commands = append(commands, pipe.HGetAll(ctx, fmt.Sprintf("%s%d:%d", groupHistoryPrefix, step, bucket)))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	since, err := sinceCmd.Int64()
	if err != nil {
		return nil, err
	}
	type gap struct{ start, end int64 }
	gaps := make([]gap, 0, len(gapCmd.Val()))
	for _, z := range gapCmd.Val() {
		first, err := strconv.ParseInt(fmt.Sprint(z.Member), 10, 64)
		if err != nil {
			return nil, err
		}
		gaps = append(gaps, gap{first, int64(z.Score)})
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, platform, status FROM groups WHERE deleted_at IS NULL ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[int64]*service.GroupRealtimeHistoryRow{}
	for rows.Next() {
		row := &service.GroupRealtimeHistoryRow{}
		if err := rows.Scan(&row.GroupID, &row.GroupName, &row.Platform, &row.Status); err != nil {
			return nil, err
		}
		byID[row.GroupID] = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := &service.GroupRealtimeHistory{StartTime: start, EndTime: end, CollectingSince: time.Unix(since, 0).UTC(), BucketSeconds: step, GroupID: groupID, Groups: []service.GroupRealtimeHistoryRow{}, Points: []service.GroupRealtimeHistoryPoint{}}
	seriesByID := map[int64][]service.GroupRealtimeHistoryPoint{}
	for i, cmd := range commands {
		bucket := start.Add(time.Duration(i*step) * time.Second)
		point := service.GroupRealtimeHistoryPoint{BucketStart: bucket, Partial: bucket.Unix() < since}
		for _, g := range gaps {
			if g.start < bucket.Unix()+int64(step) && g.end >= bucket.Unix() {
				point.Partial = true
				break
			}
		}
		// Missing coverage must not become zero traffic or a healthy success rate.
		if point.Partial {
			result.Partial = true
			result.Points = append(result.Points, point)
			continue
		}
		result.CoveredSeconds += step
		for field, value := range cmd.Val() {
			parts := strings.SplitN(field, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid group history field")
			}
			id, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				return nil, err
			}
			count, err := strconv.ParseInt(value, 10, 64)
			if err != nil || count < 0 {
				return nil, fmt.Errorf("invalid group history counter")
			}
			row := byID[id]
			if row == nil {
				row = &service.GroupRealtimeHistoryRow{GroupID: id, Status: "unknown"}
				byID[id] = row
			}
			series := seriesByID[id]
			if series == nil {
				series = make([]service.GroupRealtimeHistoryPoint, len(commands))
				seriesByID[id] = series
			}
			groupPoint := &series[i]
			selected := groupID == nil || *groupID == id
			switch parts[1] {
			case "started":
				row.Started += count
				groupPoint.Started += count
				if selected {
					point.Started += count
				}
			case "success":
				row.Success += count
				groupPoint.Success += count
				if selected {
					point.Success += count
				}
			case "failed":
				row.Failed += count
				groupPoint.Failed += count
				if selected {
					point.Failed += count
				}
			case "cancelled":
				row.Cancelled += count
				groupPoint.Cancelled += count
				if selected {
					point.Cancelled += count
				}
			}
		}
		rpm := float64(point.Started) * 60 / float64(step)
		point.RPM = &rpm
		point.SuccessRate = groupHistoryRate(point.Success, point.Failed)
		result.Points = append(result.Points, point)
	}
	for _, row := range byID {
		row.Points = seriesByID[row.GroupID]
		if row.Points == nil {
			row.Points = make([]service.GroupRealtimeHistoryPoint, len(result.Points))
		}
		for i, bucket := range result.Points {
			point := &row.Points[i]
			point.BucketStart, point.Partial = bucket.BucketStart, bucket.Partial
			if !point.Partial {
				rpm := float64(point.Started) * 60 / float64(step)
				point.RPM = &rpm
				point.SuccessRate = groupHistoryRate(point.Success, point.Failed)
			}
		}
		if result.CoveredSeconds > 0 {
			rpm := float64(row.Started) * 60 / float64(result.CoveredSeconds)
			row.RPM = &rpm
		}
		row.SuccessRate = groupHistoryRate(row.Success, row.Failed)
		result.Groups = append(result.Groups, *row)
	}
	sort.Slice(result.Groups, func(i, j int) bool {
		if result.Groups[i].Started != result.Groups[j].Started {
			return result.Groups[i].Started > result.Groups[j].Started
		}
		return result.Groups[i].GroupID < result.Groups[j].GroupID
	})
	return result, nil
}

func groupHistoryRate(success, failed int64) *float64 {
	if success+failed == 0 {
		return nil
	}
	rate := 100 * float64(success) / float64(success+failed)
	return &rate
}

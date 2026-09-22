package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const groupRealtimePrefix = "ops:group-realtime:v1:"

// Redis time keeps counters from different server instances in the same window.
// No request IDs, credentials, prompts, or response bodies are retained.
const groupRealtimeIncrementLua = `
local now = redis.call('TIME')
local key = ARGV[1] .. now[1]
redis.call('HINCRBY', key, ARGV[2], 1)
redis.call('EXPIRE', key, 120)
redis.call('SET', ARGV[1] .. 'since', now[1], 'NX')
local metric = ARGV[2]
-- History needs bounded outcome totals, not per-reason fields.
metric = string.gsub(metric, ':failed:.*$', ':failed')
for _, step in ipairs({60, 300}) do
 local bucket = math.floor(tonumber(now[1]) / step) * step
 local historyKey = ARGV[3] .. step .. ':' .. bucket
 redis.call('HINCRBY', historyKey, metric, 1)
 redis.call('EXPIRE', historyKey, 90000)
end
redis.call('SET', ARGV[3] .. 'since', now[1], 'NX')
return 1
`

var groupRealtimeIncrement = redis.NewScript(groupRealtimeIncrementLua)

// A timed-out increment may already have reached Redis. Never retry it blindly.
type groupRealtimeCounterCommand struct{ *redis.Cmd }

func (*groupRealtimeCounterCommand) NoRetry() bool { return true }

type groupRealtimeCache struct {
	rdb      *redis.Client
	writer   *redis.Client
	db       *sql.DB
	gap      atomic.Bool
	gapSince atomic.Int64
}

func NewGroupRealtimeStore(rdb *redis.Client, db *sql.DB) service.GroupRealtimeStore {
	return &groupRealtimeCache{rdb: rdb, writer: rdb.WithTimeout(100 * time.Millisecond), db: db}
}

func (r *groupRealtimeCache) Increment(ctx context.Context, groupID int64, metric string) error {
	field := fmt.Sprintf("%d:%s", groupID, metric)
	cmd := &groupRealtimeCounterCommand{redis.NewCmd(ctx, "evalsha", groupRealtimeIncrement.Hash(), 0, groupRealtimePrefix, field, groupHistoryPrefix)}
	err := r.writer.Process(ctx, cmd)
	if redis.HasErrorPrefix(err, "NOSCRIPT") {
		// NOSCRIPT guarantees the increment was not executed, so this fallback is safe.
		cmd = &groupRealtimeCounterCommand{redis.NewCmd(ctx, "eval", groupRealtimeIncrementLua, 0, groupRealtimePrefix, field, groupHistoryPrefix)}
		err = r.writer.Process(ctx, cmd)
	}
	if err != nil {
		r.gapSince.CompareAndSwap(0, time.Now().UnixMilli())
		r.gap.Store(true)
		return err
	}
	return r.publishGap(ctx)
}

func (r *groupRealtimeCache) publishGap(ctx context.Context) error {
	if !r.gap.CompareAndSwap(true, false) {
		return nil
	}
	// After recovery share the full historical gap, not just the current minute.
	since := r.gapSince.Swap(0)
	elapsed := int64(1)
	if since > 0 {
		elapsed += (time.Now().UnixMilli() - since) / 1000
	}
	if err := r.writer.Eval(ctx, groupHistoryGapLua, nil, groupRealtimePrefix, groupHistoryPrefix, elapsed).Err(); err != nil {
		r.gapSince.CompareAndSwap(0, since)
		r.gap.Store(true)
		return err
	}
	return nil
}

func (r *groupRealtimeCache) Snapshot(ctx context.Context) (*service.GroupRealtimeSnapshot, error) {
	if err := r.publishGap(ctx); err != nil {
		return nil, err
	}
	now, err := r.rdb.Time(ctx).Result()
	if err != nil {
		return nil, err
	}
	end := now.UTC().Truncate(time.Second)
	start := end.Add(-time.Minute)
	// Initialize even on an idle installation so the UI can report warm-up.
	if err := r.rdb.SetNX(ctx, groupRealtimePrefix+"since", now.Unix(), 0).Err(); err != nil {
		return nil, err
	}
	pipe := r.rdb.Pipeline()
	sinceCmd := pipe.Get(ctx, groupRealtimePrefix+"since")
	gapCmd := pipe.Exists(ctx, groupRealtimePrefix+"gap")
	commands := make([]*redis.MapStringStringCmd, 0, 60)
	for sec := start.Unix(); sec < end.Unix(); sec++ {
		commands = append(commands, pipe.HGetAll(ctx, groupRealtimePrefix+strconv.FormatInt(sec, 10)))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	since, err := sinceCmd.Int64()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, platform, status FROM groups WHERE deleted_at IS NULL ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[int64]*service.GroupRealtimeRow{}
	for rows.Next() {
		row := &service.GroupRealtimeRow{Failures: map[string]int64{}}
		if err := rows.Scan(&row.GroupID, &row.GroupName, &row.Platform, &row.Status); err != nil {
			return nil, err
		}
		byID[row.GroupID] = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, command := range commands {
		for field, value := range command.Val() {
			parts := strings.SplitN(field, ":", 3)
			if len(parts) < 2 {
				continue
			}
			id, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				return nil, err
			}
			count, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, err
			}
			row := byID[id]
			if row == nil {
				row = &service.GroupRealtimeRow{GroupID: id, Status: "unknown", Failures: map[string]int64{}}
				byID[id] = row
			}
			switch parts[1] {
			case "started":
				row.RPM += count
			case "success":
				row.Success += count
			case "cancelled":
				row.Cancelled += count
			case "failed":
				row.Failed += count
				reason := "other"
				if len(parts) == 3 {
					reason = parts[2]
				}
				row.Failures[reason] += count
			}
		}
	}
	result := &service.GroupRealtimeSnapshot{StartTime: start, EndTime: end, CollectingSince: time.Unix(since, 0).UTC(), Partial: since > start.Unix() || gapCmd.Val() > 0, Groups: []service.GroupRealtimeRow{}}
	for _, row := range byID {
		if total := row.Success + row.Failed; total > 0 {
			rate := 100 * float64(row.Success) / float64(total)
			row.SuccessRate = &rate
		}
		result.Groups = append(result.Groups, *row)
	}
	sort.Slice(result.Groups, func(i, j int) bool {
		if result.Groups[i].RPM != result.Groups[j].RPM {
			return result.Groups[i].RPM > result.Groups[j].RPM
		}
		return result.Groups[i].GroupID < result.Groups[j].GroupID
	})
	return result, nil
}

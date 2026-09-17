package repository

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGroupRealtimeRollingWindowAndEmptyGroups(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := NewGroupRealtimeStore(rdb, db)
	ctx := context.Background()
	end := time.Date(2026, 9, 17, 12, 1, 25, 0, time.UTC)
	// Cross the wall-clock minute boundary, and include exactly the start second.
	for _, event := range []struct {
		offset int
		metric string
	}{{-61, "started"}, {-60, "started"}, {-35, "started"}, {-2, "success"}, {-1, "failed:timeout"}, {-1, "cancelled"}, {0, "success"}} {
		mr.SetTime(end.Add(time.Duration(event.offset) * time.Second))
		require.NoError(t, store.Increment(ctx, 7, event.metric))
	}
	mr.SetTime(end)
	require.Positive(t, mr.TTL(groupRealtimePrefix+strconv.FormatInt(end.Unix()-60, 10)))
	mock.ExpectQuery("SELECT id, name, platform, status FROM groups").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "platform", "status"}).AddRow(7, "Busy", "openai", "active").AddRow(8, "Idle", "claude", "inactive"))
	snapshot, err := store.Snapshot(ctx)
	require.NoError(t, err)
	require.Equal(t, end.Add(-time.Minute), snapshot.StartTime)
	require.Len(t, snapshot.Groups, 2)
	busy, idle := snapshot.Groups[0], snapshot.Groups[1]
	require.EqualValues(t, 2, busy.RPM)
	require.EqualValues(t, 1, busy.Success)
	require.EqualValues(t, 1, busy.Failed)
	require.EqualValues(t, 1, busy.Cancelled)
	require.Equal(t, 50.0, *busy.SuccessRate)
	require.EqualValues(t, 1, busy.Failures["timeout"])
	require.Nil(t, idle.SuccessRate)
	require.Zero(t, idle.RPM)
	require.False(t, snapshot.Partial)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupRealtimeSharedGapAfterRecovery(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = rdb.Close() })
	store := NewGroupRealtimeStore(rdb, nil)
	ctx := context.Background()
	mr.SetError("ERR temporarily unavailable")
	require.Error(t, store.Increment(ctx, 1, "started"))
	mr.SetError("")
	require.NoError(t, store.Increment(ctx, 1, "success"))
	require.True(t, mr.Exists(groupRealtimePrefix+"gap"))
	require.Equal(t, time.Minute, mr.TTL(groupRealtimePrefix+"gap"))
}

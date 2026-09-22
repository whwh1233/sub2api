package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func historyTestStore(t *testing.T) (*groupRealtimeCache, *miniredis.Miniredis, sqlmock.Sqlmock) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close(); _ = db.Close() })
	return NewGroupRealtimeStore(rdb, db).(*groupRealtimeCache), mr, mock
}
func expectHistoryGroups(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT id, name, platform, status FROM groups").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "platform", "status"}).AddRow(1, "Alpha", "openai", "active").AddRow(2, "Beta", "claude", "active").AddRow(3, "Idle", "openai", "inactive"))
}

func TestGroupHistorySharedCountersBoundariesAndWeightedRates(t *testing.T) {
	store, mr, mock := historyTestStore(t)
	ctx := context.Background()
	end := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	start := end.Add(-15 * time.Minute)
	mr.Set(groupHistoryPrefix+"since", fmt.Sprint(start.Unix()))
	// Exactly the start is included; exactly the end and the preceding minute aren't.
	for _, event := range []struct {
		at     time.Time
		group  int64
		metric string
	}{
		{start.Add(-time.Second), 1, "started"}, {start, 1, "started"}, {start, 1, "success"},
		{start.Add(time.Minute), 1, "failed:timeout"}, {start, 1, "cancelled"},
		{start, 2, "started"}, {start, 2, "started"}, {start, 2, "success"}, {start, 2, "success"},
		{start, 0, "started"}, {start, 0, "failed:authentication"},
		{end, 1, "started"}, {end, 1, "success"},
	} {
		mr.SetTime(event.at)
		require.NoError(t, store.Increment(ctx, event.group, event.metric))
	}
	mr.SetTime(end.Add(40 * time.Second))
	expectHistoryGroups(mock)
	result, err := store.readRedisHistory(ctx, 15, nil)
	require.NoError(t, err)
	require.Equal(t, end, result.EndTime)
	require.Equal(t, 900, result.CoveredSeconds)
	require.False(t, result.Partial)
	require.Len(t, result.Points, 15)
	for i, total := range result.Points {
		var started, success, failed int64
		for _, group := range result.Groups {
			require.Len(t, group.Points, len(result.Points))
			point := group.Points[i]
			require.Equal(t, total.BucketStart, point.BucketStart)
			started += point.Started
			success += point.Success
			failed += point.Failed
		}
		require.Equal(t, total.Started, started)
		require.Equal(t, total.Success, success)
		require.Equal(t, total.Failed, failed)
	}
	require.EqualValues(t, 4, result.Points[0].Started)
	require.Equal(t, 75.0, *result.Points[0].SuccessRate)
	require.EqualValues(t, 1, result.Points[1].Failed)
	require.Nil(t, result.Points[2].SuccessRate)
	require.Equal(t, 0.0, *result.Points[2].RPM)
	require.EqualValues(t, 2, result.Groups[0].GroupID)
	require.InDelta(t, 2.0/15, *result.Groups[0].RPM, 0.00001)
	for _, row := range result.Groups {
		if row.GroupID == 1 {
			require.Equal(t, 50.0, *row.SuccessRate)
			require.EqualValues(t, 1, row.Started)
			require.EqualValues(t, 1, row.Cancelled)
		}
		if row.GroupID == 3 {
			require.Nil(t, row.SuccessRate)
			require.Equal(t, 0.0, *row.RPM)
		}
	}
	// Filter only the chart; keep the full per-group summary for comparison.
	id := int64(1)
	expectHistoryGroups(mock)
	selected, err := store.readRedisHistory(ctx, 15, &id)
	require.NoError(t, err)
	require.EqualValues(t, 1, selected.Points[0].Started)
	require.Equal(t, 100.0, *selected.Points[0].SuccessRate)
	require.Len(t, selected.Groups, 4)
	require.Equal(t, 25*time.Hour, mr.TTL(fmt.Sprintf("%s60:%d", groupHistoryPrefix, start.Unix())))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupHistoryWarmupAndCollectionGapsRemainNull(t *testing.T) {
	store, mr, mock := historyTestStore(t)
	ctx := context.Background()
	end := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	mr.SetTime(end)
	// Starting mid-minute excludes the first partial bucket.
	mr.Set(groupHistoryPrefix+"since", fmt.Sprint(end.Add(-150*time.Second).Unix()))
	// Gap crossing the last two completed minutes excludes both.
	require.NoError(t, store.rdb.ZAdd(ctx, groupHistoryPrefix+"gaps", redis.Z{Score: float64(end.Add(-50 * time.Second).Unix()), Member: fmt.Sprint(end.Add(-70 * time.Second).Unix())}).Err())
	expectHistoryGroups(mock)
	result, err := store.readRedisHistory(ctx, 15, nil)
	require.NoError(t, err)
	require.True(t, result.Partial)
	require.Zero(t, result.CoveredSeconds)
	for _, point := range result.Points {
		require.True(t, point.Partial)
		require.Nil(t, point.RPM)
		require.Nil(t, point.SuccessRate)
	}
	for _, row := range result.Groups {
		require.Nil(t, row.RPM)
		require.Nil(t, row.SuccessRate)
		for _, point := range row.Points {
			require.True(t, point.Partial)
			require.Nil(t, point.RPM)
			require.Nil(t, point.SuccessRate)
		}
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupHistoryFiveMinuteBucketsAndReadFailures(t *testing.T) {
	store, mr, mock := historyTestStore(t)
	ctx := context.Background()
	end := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	mr.Set(groupHistoryPrefix+"since", fmt.Sprint(end.Add(-24*time.Hour).Unix()))
	mr.SetTime(end.Add(-time.Minute))
	require.NoError(t, store.Increment(ctx, 1, "started"))
	require.NoError(t, store.Increment(ctx, 1, "success"))
	mr.SetTime(end.Add(2 * time.Minute))
	expectHistoryGroups(mock)
	result, err := store.readRedisHistory(ctx, 1440, nil)
	require.NoError(t, err)
	require.Equal(t, 300, result.BucketSeconds)
	require.Equal(t, end, result.EndTime)
	require.Len(t, result.Points, 288)
	require.Equal(t, 0.2, *result.Points[287].RPM)
	require.Equal(t, 100.0, *result.Points[287].SuccessRate)
	_, err = store.readRedisHistory(ctx, 999999, nil)
	require.Error(t, err)
	mr.SetError("ERR offline")
	_, err = store.readRedisHistory(ctx, 15, nil)
	require.Error(t, err)
	mr.SetError("")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupHistoryRecoveryPublishesHistoricalGap(t *testing.T) {
	store, mr, mock := historyTestStore(t)
	ctx := context.Background()
	end := time.Date(2026, 9, 22, 12, 1, 0, 0, time.UTC)
	mr.SetTime(end.Add(-time.Second))
	mr.Set(groupHistoryPrefix+"since", fmt.Sprint(end.Add(-time.Hour).Unix()))
	mr.SetError("ERR unavailable")
	require.Error(t, store.Increment(ctx, 1, "started"))
	store.gapSince.Store(time.Now().Add(-2 * time.Minute).UnixMilli())
	mr.SetError("")
	require.NoError(t, store.Increment(ctx, 1, "success"))
	mr.SetTime(end)
	expectHistoryGroups(mock)
	result, err := store.readRedisHistory(ctx, 15, nil)
	require.NoError(t, err)
	require.True(t, result.Partial)
	require.True(t, result.Points[14].Partial)
	require.True(t, result.Points[13].Partial)
	require.True(t, result.Points[12].Partial)
	require.Equal(t, 12*60, result.CoveredSeconds)
	require.NoError(t, mock.ExpectationsWereMet())
}

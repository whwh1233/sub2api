package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type historyTestStore struct {
	realtimeTestStore
	calls int
}

func (s *historyTestStore) History(_ context.Context, minutes int, id *int64) (*GroupRealtimeHistory, error) {
	s.calls++
	return &GroupRealtimeHistory{GroupID: id, BucketSeconds: 60, EndTime: time.Now()}, nil
}
func TestGroupHistoryValidationAndCacheSeparatesSelections(t *testing.T) {
	store := &historyTestStore{}
	svc := &OpsService{groupRealtime: NewGroupRealtimeMonitor(store)}
	ctx := context.Background()
	_, err := svc.GetGroupRealtimeHistory(ctx, 16, nil)
	require.Error(t, err)
	negative := int64(-1)
	_, err = svc.GetGroupRealtimeHistory(ctx, 60, &negative)
	require.Error(t, err)
	require.Zero(t, store.calls)
	a, err := svc.GetGroupRealtimeHistory(ctx, 60, nil)
	require.NoError(t, err)
	b, err := svc.GetGroupRealtimeHistory(ctx, 60, nil)
	require.NoError(t, err)
	require.Same(t, a, b)
	require.Equal(t, 1, store.calls)
	id := int64(0)
	_, err = svc.GetGroupRealtimeHistory(ctx, 60, &id)
	require.NoError(t, err)
	require.Equal(t, 2, store.calls)
	_, err = svc.GetGroupRealtimeHistory(ctx, 15, &id)
	require.NoError(t, err)
	require.Equal(t, 3, store.calls)
	_, err = svc.GetGroupRealtimeHistory(ctx, 10080, nil)
	require.NoError(t, err)
	_, err = svc.GetGroupRealtimeHistory(ctx, 10081, nil)
	require.Error(t, err)
	svc.groupRealtime = nil
	_, err = svc.GetGroupRealtimeHistory(ctx, 60, nil)
	require.Error(t, err)
}

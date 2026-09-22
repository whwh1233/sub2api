package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type realtimeTestStore struct {
	calls int
	err   error
}

func (s *realtimeTestStore) Snapshot(context.Context) (*GroupRealtimeSnapshot, error) {
	s.calls++
	return &GroupRealtimeSnapshot{EndTime: time.Now()}, s.err
}
func TestGroupRealtimeReadOnlyCache(t *testing.T) {
	store := &realtimeTestStore{}
	svc := &OpsService{groupRealtime: NewGroupRealtimeMonitor(store)}
	a, err := svc.GetGroupRealtime(context.Background())
	require.NoError(t, err)
	b, err := svc.GetGroupRealtime(context.Background())
	require.NoError(t, err)
	require.Same(t, a, b)
	require.Equal(t, 1, store.calls)
	svc.groupRealtime.cachedAt = time.Time{}
	store.err = errors.New("database unavailable")
	_, err = svc.GetGroupRealtime(context.Background())
	require.Error(t, err)
}

package service

import (
	"context"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// GroupRealtimeStore reads existing PostgreSQL logs; requests do not write monitoring counters.
type GroupRealtimeStore interface {
	Snapshot(context.Context) (*GroupRealtimeSnapshot, error)
}

type GroupRealtimeRow struct {
	GroupID     int64            `json:"group_id"`
	GroupName   string           `json:"group_name"`
	Platform    string           `json:"platform"`
	Status      string           `json:"status"`
	RPM         int64            `json:"rpm"`
	Success     int64            `json:"success"`
	Failed      int64            `json:"failed"`
	Cancelled   int64            `json:"cancelled"`
	SuccessRate *float64         `json:"success_rate"`
	Failures    map[string]int64 `json:"failures"`
}

type GroupRealtimeSnapshot struct {
	StartTime       time.Time          `json:"start_time"`
	EndTime         time.Time          `json:"end_time"`
	CollectingSince time.Time          `json:"collecting_since"`
	Partial         bool               `json:"partial"`
	Groups          []GroupRealtimeRow `json:"groups"`
}

type GroupRealtimeMonitor struct {
	store          GroupRealtimeStore
	cacheMu        sync.Mutex
	cached         *GroupRealtimeSnapshot
	cachedAt       time.Time
	historyMu      sync.Mutex
	history        *GroupRealtimeHistory
	historyMinutes int
	historyAt      time.Time
}

func NewGroupRealtimeMonitor(store GroupRealtimeStore) *GroupRealtimeMonitor {
	return &GroupRealtimeMonitor{store: store}
}

func (s *OpsService) GetGroupRealtime(ctx context.Context) (*GroupRealtimeSnapshot, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if s.groupRealtime == nil || s.groupRealtime.store == nil {
		return nil, infraerrors.ServiceUnavailable("GROUP_REALTIME_UNAVAILABLE", "Group realtime collector unavailable")
	}
	m := s.groupRealtime
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	if m.cached != nil && time.Since(m.cachedAt) < 3*time.Second {
		return m.cached, nil
	}
	result, err := m.store.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	m.cached, m.cachedAt = result, time.Now()
	return result, nil
}

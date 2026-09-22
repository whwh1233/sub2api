package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// History and realtime use the same persisted usage/error-log definition.
type GroupRealtimeHistoryPersister interface {
	PersistHistory(context.Context) error
}

type GroupRealtimeHistoryStore interface {
	History(context.Context, int, *int64) (*GroupRealtimeHistory, error)
}

type GroupRealtimeHistoryPoint struct {
	BucketStart time.Time `json:"bucket_start"`
	RPM         *float64  `json:"rpm"`
	SuccessRate *float64  `json:"success_rate"`
	Started     int64     `json:"started"`
	Success     int64     `json:"success"`
	Failed      int64     `json:"failed"`
	Cancelled   int64     `json:"cancelled"`
	Partial     bool      `json:"partial"`
}

type GroupRealtimeHistoryRow struct {
	Points      []GroupRealtimeHistoryPoint `json:"points"`
	GroupID     int64                       `json:"group_id"`
	GroupName   string                      `json:"group_name"`
	Platform    string                      `json:"platform"`
	Status      string                      `json:"status"`
	Started     int64                       `json:"started"`
	Success     int64                       `json:"success"`
	Failed      int64                       `json:"failed"`
	Cancelled   int64                       `json:"cancelled"`
	RPM         *float64                    `json:"rpm"`
	SuccessRate *float64                    `json:"success_rate"`
}

type GroupRealtimeHistory struct {
	StartTime       time.Time                   `json:"start_time"`
	EndTime         time.Time                   `json:"end_time"`
	CollectingSince time.Time                   `json:"collecting_since"`
	BucketSeconds   int                         `json:"bucket_seconds"`
	CoveredSeconds  int                         `json:"covered_seconds"`
	Partial         bool                        `json:"partial"`
	GroupID         *int64                      `json:"group_id"`
	Groups          []GroupRealtimeHistoryRow   `json:"groups"`
	Points          []GroupRealtimeHistoryPoint `json:"points"`
}

func (s *OpsService) GetGroupRealtimeHistory(ctx context.Context, minutes int, groupID *int64) (*GroupRealtimeHistory, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	switch minutes {
	case 15, 60, 360, 1440, 10080:
	default:
		return nil, infraerrors.BadRequest("GROUP_HISTORY_RANGE_INVALID", "window_minutes must be 15, 60, 360, 1440, or 10080")
	}
	if groupID != nil && *groupID < 0 {
		return nil, infraerrors.BadRequest("GROUP_HISTORY_GROUP_INVALID", "group_id must be non-negative")
	}
	if s.groupRealtime == nil {
		return nil, infraerrors.ServiceUnavailable("GROUP_HISTORY_UNAVAILABLE", "Group history collector unavailable")
	}
	m := s.groupRealtime
	store, ok := m.store.(GroupRealtimeHistoryStore)
	if !ok {
		return nil, infraerrors.ServiceUnavailable("GROUP_HISTORY_UNAVAILABLE", "Group history collector unavailable")
	}
	m.historyMu.Lock()
	defer m.historyMu.Unlock()
	sameGroup := m.history != nil && ((groupID == nil && m.history.GroupID == nil) || (groupID != nil && m.history.GroupID != nil && *groupID == *m.history.GroupID))
	if sameGroup && m.historyMinutes == minutes && time.Since(m.historyAt) < 15*time.Second {
		return m.history, nil
	}
	result, err := store.History(ctx, minutes, groupID)
	if err != nil {
		return nil, err
	}
	m.history, m.historyMinutes, m.historyAt = result, minutes, time.Now()
	return result, nil
}

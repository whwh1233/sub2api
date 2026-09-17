package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

// GroupRealtimeStore uses shared, expiring counters rather than sampled logs.
// Metrics are started, success, cancelled, or failed:<bounded reason>.
type GroupRealtimeStore interface {
	Increment(context.Context, int64, string) error
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
	store    GroupRealtimeStore
	failedAt atomic.Int64
	cacheMu  sync.Mutex
	cached   *GroupRealtimeSnapshot
	cachedAt time.Time
}

func NewGroupRealtimeMonitor(store GroupRealtimeStore) *GroupRealtimeMonitor {
	return &GroupRealtimeMonitor{store: store}
}

func (m *GroupRealtimeMonitor) record(groupID int64, metric string) {
	if m == nil || m.store == nil {
		return
	}
	// Accounting failure must never reject a user request. Bound the extra work
	// and expose gaps instead of displaying a misleading healthy zero.
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if err := m.store.Increment(ctx, groupID, metric); err != nil {
		m.failedAt.Store(time.Now().Unix())
	}
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
	if m.cached != nil && time.Since(m.cachedAt) < 3*time.Second && m.failedAt.Load() < m.cached.StartTime.Unix() {
		return m.cached, nil
	}
	result, err := m.store.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	if s.groupRealtime.failedAt.Load() >= result.StartTime.Unix() {
		result.Partial = true
	}
	m.cached, m.cachedAt = result, time.Now()
	return result, nil
}

const groupRealtimeObserverKey = "group_realtime_observer"

// GroupRealtimeObserver owns a logical HTTP request or the active WS turn.
// Its group is frozen before composite routing rewrites APIKey.GroupID.
type GroupRealtimeObserver struct {
	mu             sync.Mutex
	monitor        *GroupRealtimeMonitor
	groupID        int64
	bound          bool
	websocket      bool
	active         bool
	hadTurn        bool
	pendingFailure string
}

func (s *OpsService) BeginGroupRealtime(c *gin.Context, websocket bool) *GroupRealtimeObserver {
	if s == nil || s.groupRealtime == nil || !s.IsMonitoringEnabled(c.Request.Context()) {
		return nil
	}
	o := &GroupRealtimeObserver{monitor: s.groupRealtime, websocket: websocket}
	c.Set(groupRealtimeObserverKey, o)
	return o
}

func groupRealtimeObserver(c *gin.Context) *GroupRealtimeObserver {
	if c == nil {
		return nil
	}
	v, _ := c.Get(groupRealtimeObserverKey)
	o, _ := v.(*GroupRealtimeObserver)
	return o
}

// BindGroupRealtime is called immediately after key lookup, including keys
// subsequently rejected for balance, group access, or rate limits.
func BindGroupRealtime(c *gin.Context, key *APIKey) {
	o := groupRealtimeObserver(c)
	if o == nil || key == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.bound {
		return
	}
	o.bound = true
	if key.GroupID != nil {
		o.groupID = *key.GroupID
	}
	if !o.websocket {
		o.startLocked()
	}
}

func (o *GroupRealtimeObserver) startLocked() {
	if o.active {
		return
	}
	o.active = true
	o.pendingFailure = ""
	o.monitor.record(o.groupID, "started")
}

func StartGroupRealtimeTurn(c *gin.Context) {
	o := groupRealtimeObserver(c)
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.hadTurn = true
	o.startLocked() // retrying the current turn must not increment again
}

// FinishGroupRealtimeTurn defers retryable errors until the retry succeeds or
// the connection handler returns. Successful turns finish immediately.
func FinishGroupRealtimeTurn(c *gin.Context, outcome string, retryable bool) {
	o := groupRealtimeObserver(c)
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.active {
		return
	}
	if retryable {
		o.pendingFailure = outcome
		return
	}
	o.monitor.record(o.groupID, outcome)
	o.active = false
	o.pendingFailure = ""
}

func (o *GroupRealtimeObserver) Finish(outcome string) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.websocket && o.hadTurn && !o.active {
		return
	}
	if o.websocket && !o.hadTurn && (outcome == "success" || outcome == "cancelled") {
		return
	}
	if o.websocket && o.active && outcome == "success" {
		outcome = "failed:transport_or_stream"
	}
	if !o.active {
		o.startLocked()
	}
	if o.pendingFailure != "" && outcome != "cancelled" {
		outcome = o.pendingFailure
	}
	o.monitor.record(o.groupID, outcome)
	o.active = false
}

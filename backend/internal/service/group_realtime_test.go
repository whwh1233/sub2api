package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type realtimeEvent struct {
	group  int64
	metric string
}
type realtimeTestStore struct {
	events []realtimeEvent
	err    error
}

func (s *realtimeTestStore) Increment(_ context.Context, group int64, metric string) error {
	s.events = append(s.events, realtimeEvent{group, metric})
	return s.err
}
func (s *realtimeTestStore) Snapshot(context.Context) (*GroupRealtimeSnapshot, error) {
	return nil, nil
}

func newRealtimeTestObserver(t *testing.T, ws bool) (*gin.Context, *GroupRealtimeObserver, *realtimeTestStore) {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	store := &realtimeTestStore{}
	svc := &OpsService{groupRealtime: NewGroupRealtimeMonitor(store)}
	return c, svc.BeginGroupRealtime(c, ws), store
}

func TestGroupRealtimeHTTPFreezesEntryGroupAndCountsBeforeCompletion(t *testing.T) {
	c, observer, store := newRealtimeTestObserver(t, false)
	group := int64(7)
	key := &APIKey{GroupID: &group}
	BindGroupRealtime(c, key)
	require.Equal(t, []realtimeEvent{{7, "started"}}, store.events)
	group = 12 // composite routing / failover must not change attribution
	BindGroupRealtime(c, key)
	observer.Finish("success")
	require.Equal(t, []realtimeEvent{{7, "started"}, {7, "success"}}, store.events)
}

func TestGroupRealtimeWSRetriesTurnsAndIdleClose(t *testing.T) {
	c, observer, store := newRealtimeTestObserver(t, true)
	group := int64(3)
	BindGroupRealtime(c, &APIKey{GroupID: &group})
	require.Empty(t, store.events, "handshake is not an inference request")
	StartGroupRealtimeTurn(c)
	FinishGroupRealtimeTurn(c, "failed:timeout", true)
	StartGroupRealtimeTurn(c)
	FinishGroupRealtimeTurn(c, "success", false)
	StartGroupRealtimeTurn(c)
	FinishGroupRealtimeTurn(c, "cancelled", false)
	observer.Finish("success")
	require.Equal(t, []realtimeEvent{{3, "started"}, {3, "success"}, {3, "started"}, {3, "cancelled"}}, store.events)
}

func TestGroupRealtimeWSExhaustedRetryAndIncompleteTurn(t *testing.T) {
	for _, retry := range []bool{false, true} {
		c, observer, store := newRealtimeTestObserver(t, true)
		StartGroupRealtimeTurn(c)
		if retry {
			FinishGroupRealtimeTurn(c, "failed:timeout", true)
		}
		observer.Finish("success")
		want := "failed:transport_or_stream"
		if retry {
			want = "failed:timeout"
		}
		require.Equal(t, []realtimeEvent{{0, "started"}, {0, want}}, store.events)
	}
}

func TestGroupRealtimeUnknownGroupAndCollectorFailure(t *testing.T) {
	_, observer, store := newRealtimeTestObserver(t, false)
	store.err = errors.New("redis unavailable")
	observer.Finish("failed:authentication")
	require.Equal(t, []realtimeEvent{{0, "started"}, {0, "failed:authentication"}}, store.events)
	require.Positive(t, observer.monitor.failedAt.Load())
}

func TestGroupRealtimeCancellationDuringRetry(t *testing.T) {
	c, observer, store := newRealtimeTestObserver(t, true)
	StartGroupRealtimeTurn(c)
	FinishGroupRealtimeTurn(c, "failed:timeout", true)
	observer.Finish("cancelled")
	require.Equal(t, []realtimeEvent{{0, "started"}, {0, "cancelled"}}, store.events)
}

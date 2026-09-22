package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

type groupHistoryPersisterStub struct{ calls int }

func (p *groupHistoryPersisterStub) PersistHistory(ctx context.Context) error {
	p.calls++
	_, ok := ctx.Deadline()
	if !ok {
		panic("missing persistence deadline")
	}
	return nil
}
func TestGroupHistoryCollectorRespectsMonitoringSwitch(t *testing.T) {
	p := &groupHistoryPersisterStub{}
	c := &OpsMetricsCollector{groupHistory: p}
	c.collectGroupHistoryOnce()
	require.Equal(t, 1, p.calls)
	c.cfg = &config.Config{}
	c.collectGroupHistoryOnce()
	require.Equal(t, 1, p.calls)
}

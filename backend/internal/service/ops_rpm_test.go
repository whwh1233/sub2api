package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPickOpsRPMBuckets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		window     time.Duration
		wantSource int
		wantOutput int
	}{
		{name: "six hours stays exact", window: 6 * time.Hour, wantSource: 60, wantOutput: 60},
		{name: "one day uses five minute display", window: 24 * time.Hour, wantSource: 60, wantOutput: 300},
		{name: "seven days uses thirty minute display", window: 7 * 24 * time.Hour, wantSource: 60, wantOutput: 1800},
		{name: "thirty days uses retained five minute source", window: 30 * 24 * time.Hour, wantSource: 300, wantOutput: 7200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source, output := pickOpsRPMBuckets(tt.window)
			require.Equal(t, tt.wantSource, source)
			require.Equal(t, tt.wantOutput, output)
		})
	}
}

func TestOpsServiceGetRPMTrendNormalizesFilter(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	var captured *OpsRPMTrendFilter
	repo := &opsRepoMock{GetRPMTrendFn: func(_ context.Context, filter *OpsRPMTrendFilter) (*OpsRPMTrendResponse, error) {
		copy := *filter
		captured = &copy
		return &OpsRPMTrendResponse{Series: []*OpsRPMTrendSeries{}}, nil
	}}
	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := svc.GetRPMTrend(context.Background(), &OpsRPMTrendFilter{
		StartTime: start,
		EndTime:   end,
		Dimension: OpsRPMDimensionAccount,
		TopN:      99,
	})

	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Equal(t, 60, captured.SourceBucketSeconds)
	require.Equal(t, 300, captured.OutputBucketSeconds)
	require.Equal(t, opsRPMMaxTopN, captured.TopN)
	require.Equal(t, OpsRPMDimensionAccount, result.Dimension)
	require.Equal(t, start, result.StartTime)
	require.Equal(t, end, result.EndTime)
}

func TestOpsServiceGetRPMTrendRejectsInvalidDimensionAndWindow(t *testing.T) {
	t.Parallel()
	svc := NewOpsService(&opsRepoMock{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	end := time.Now().UTC()

	_, err := svc.GetRPMTrend(context.Background(), &OpsRPMTrendFilter{
		StartTime: end.Add(-time.Hour), EndTime: end, Dimension: "secret",
	})
	require.Error(t, err)

	_, err = svc.GetRPMTrend(context.Background(), &OpsRPMTrendFilter{
		StartTime: end.Add(-31 * 24 * time.Hour), EndTime: end, Dimension: OpsRPMDimensionPlatform,
	})
	require.Error(t, err)
}

func TestOpsRPMCollectorUsesBoundedOverlapAndCompactRollup(t *testing.T) {
	t.Parallel()
	windowEnd := time.Date(2026, 8, 5, 3, 7, 0, 0, time.UTC)
	var minuteStart, minuteEnd time.Time
	type rollupCall struct {
		source, target int
		start, end     time.Time
	}
	rollups := make([]rollupCall, 0, 1)
	repo := &opsRepoMock{
		UpsertRPMMinuteMetricsFn: func(_ context.Context, startTime, endTime time.Time) error {
			minuteStart, minuteEnd = startTime, endTime
			return nil
		},
		UpsertRPMRollupFn: func(_ context.Context, source, target int, startTime, endTime time.Time) error {
			rollups = append(rollups, rollupCall{source: source, target: target, start: startTime, end: endTime})
			return nil
		},
	}
	collector := &OpsMetricsCollector{opsRepo: repo}

	require.NoError(t, collector.collectAndPersistRPM(context.Background(), windowEnd))
	require.Equal(t, windowEnd.Add(-5*time.Minute), minuteStart)
	require.Equal(t, windowEnd, minuteEnd)
	require.Equal(t, []rollupCall{{
		source: 60,
		target: 300,
		start:  time.Date(2026, 8, 5, 3, 0, 0, 0, time.UTC),
		end:    time.Date(2026, 8, 5, 3, 5, 0, 0, time.UTC),
	}}, rollups)
}

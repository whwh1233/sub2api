package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	opsRPMDefaultTopN = 10
	opsRPMMaxTopN     = 20
)

func (s *OpsService) GetRPMTrend(ctx context.Context, filter *OpsRPMTrendFilter) (*OpsRPMTrendResponse, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if s.opsRepo == nil {
		return nil, infraerrors.ServiceUnavailable("OPS_REPO_UNAVAILABLE", "Ops repository not available")
	}
	if filter == nil {
		return nil, infraerrors.BadRequest("OPS_RPM_FILTER_REQUIRED", "filter is required")
	}
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() || !filter.StartTime.Before(filter.EndTime) {
		return nil, infraerrors.BadRequest("OPS_RPM_TIME_RANGE_INVALID", "valid start_time/end_time are required")
	}
	if filter.EndTime.Sub(filter.StartTime) > 30*24*time.Hour {
		return nil, infraerrors.BadRequest("OPS_RPM_TIME_RANGE_TOO_LARGE", "max window is 30 days")
	}
	if ParseOpsRPMDimension(string(filter.Dimension)) == "" {
		return nil, infraerrors.BadRequest("OPS_RPM_DIMENSION_INVALID", "dimension must be platform, model, account, or user")
	}

	filter.SourceBucketSeconds, filter.OutputBucketSeconds = pickOpsRPMBuckets(filter.EndTime.Sub(filter.StartTime))
	if filter.TopN <= 0 {
		filter.TopN = opsRPMDefaultTopN
	}
	if filter.TopN > opsRPMMaxTopN {
		filter.TopN = opsRPMMaxTopN
	}

	result, err := s.opsRepo.GetRPMTrend(ctx, filter)
	if err != nil {
		return nil, err
	}
	result.GeneratedAt = time.Now().UTC()
	result.StartTime = filter.StartTime.UTC()
	result.EndTime = filter.EndTime.UTC()
	result.Dimension = filter.Dimension
	result.SourceBucketSeconds = filter.SourceBucketSeconds
	result.OutputBucketSeconds = filter.OutputBucketSeconds
	return result, nil
}

func pickOpsRPMBuckets(window time.Duration) (sourceSeconds, outputSeconds int) {
	switch {
	case window <= 6*time.Hour:
		return 60, 60
	case window <= 24*time.Hour:
		return 60, 300
	case window <= 7*24*time.Hour:
		return 60, 1800
	default:
		return 300, 7200
	}
}

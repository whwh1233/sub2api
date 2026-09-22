package service

import "time"

type OpsRPMDimension string

const (
	OpsRPMDimensionPlatform OpsRPMDimension = "platform"
	OpsRPMDimensionModel    OpsRPMDimension = "model"
	OpsRPMDimensionAccount  OpsRPMDimension = "account"
	OpsRPMDimensionUser     OpsRPMDimension = "user"
)

func ParseOpsRPMDimension(value string) OpsRPMDimension {
	switch OpsRPMDimension(value) {
	case OpsRPMDimensionPlatform, OpsRPMDimensionModel, OpsRPMDimensionAccount, OpsRPMDimensionUser:
		return OpsRPMDimension(value)
	default:
		return ""
	}
}

type OpsRPMTrendFilter struct {
	StartTime           time.Time
	EndTime             time.Time
	Dimension           OpsRPMDimension
	SourceBucketSeconds int
	OutputBucketSeconds int
	TopN                int
}

type OpsRPMTrendPoint struct {
	BucketStart  time.Time `json:"bucket_start"`
	SuccessCount int64     `json:"success_count"`
	ErrorCount   int64     `json:"error_count"`
	SuccessRPM   float64   `json:"success_rpm"`
	ErrorRPM     float64   `json:"error_rpm"`
	TotalRPM     float64   `json:"total_rpm"`
}

type OpsRPMTrendSeries struct {
	Key    string              `json:"key"`
	Label  string              `json:"label"`
	Points []*OpsRPMTrendPoint `json:"points"`
}

type OpsRPMTrendResponse struct {
	GeneratedAt         time.Time            `json:"generated_at"`
	StartTime           time.Time            `json:"start_time"`
	EndTime             time.Time            `json:"end_time"`
	CompleteThrough     *time.Time           `json:"complete_through,omitempty"`
	Dimension           OpsRPMDimension      `json:"dimension"`
	SourceBucketSeconds int                  `json:"source_bucket_seconds"`
	OutputBucketSeconds int                  `json:"output_bucket_seconds"`
	Series              []*OpsRPMTrendSeries `json:"series"`
}

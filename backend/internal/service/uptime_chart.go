package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type UptimeChartRepository interface {
	GetUptimeBucketRows(ctx context.Context, query UptimeChartQuery) ([]UptimeBucketRow, error)
}

type UptimeScope string

const (
	UptimeScopeUser  UptimeScope = "user"
	UptimeScopeAdmin UptimeScope = "admin"
)

type UptimeDimension string

const (
	UptimeDimensionAll    UptimeDimension = "all"
	UptimeDimensionAPIKey UptimeDimension = "api_key"
	UptimeDimensionModel  UptimeDimension = "model"
	UptimeDimensionGroup  UptimeDimension = "group"
)

type UptimeChartQuery struct {
	Scope         UptimeScope
	Dimension     UptimeDimension
	UserID        *int64
	StartTime     time.Time
	EndTime       time.Time
	Window        string
	BucketSeconds int
}

type UptimeBucketRow struct {
	Bucket  time.Time
	Key     string
	Label   string
	Success int64
	Failure int64
}

type UptimeChartResponse struct {
	Scope               UptimeScope       `json:"scope"`
	Window              string            `json:"window"`
	Dimension           UptimeDimension   `json:"dimension"`
	StartTime           time.Time         `json:"start_time"`
	EndTime             time.Time         `json:"end_time"`
	BucketSeconds       int               `json:"bucket_seconds"`
	SuccessSource       string            `json:"success_source"`
	FailureSource       string            `json:"failure_source"`
	SLAExcludesBusiness bool              `json:"sla_excludes_business_limited"`
	Series              []UptimeSeries    `json:"series"`
	Summary             UptimeSeriesStats `json:"summary"`
}

type UptimeSeries struct {
	Key     string            `json:"key"`
	Label   string            `json:"label"`
	Summary UptimeSeriesStats `json:"summary"`
	Buckets []UptimeBucket    `json:"buckets"`
}

type UptimeSeriesStats struct {
	Total        int64    `json:"total"`
	Success      int64    `json:"success"`
	Failure      int64    `json:"failure"`
	Availability *float64 `json:"availability"`
}

type UptimeBucket struct {
	Start        time.Time `json:"start"`
	Total        int64     `json:"total"`
	Success      int64     `json:"success"`
	Failure      int64     `json:"failure"`
	Availability *float64  `json:"availability"`
}

type UptimeChartService struct {
	repo        UptimeChartRepository
	settingRepo SettingRepository
	cfg         *config.Config
}

func NewUptimeChartService(repo UptimeChartRepository, settingRepo SettingRepository, cfg *config.Config) *UptimeChartService {
	return &UptimeChartService{repo: repo, settingRepo: settingRepo, cfg: cfg}
}

func (s *UptimeChartService) GetChart(ctx context.Context, input UptimeChartQuery) (*UptimeChartResponse, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("UPTIME_UNAVAILABLE", "uptime chart service is unavailable")
	}
	if err := s.requireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	query, err := normalizeUptimeQuery(input, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.GetUptimeBucketRows(ctx, query)
	if err != nil {
		return nil, err
	}
	return buildUptimeResponse(query, rows), nil
}

func (s *UptimeChartService) requireMonitoringEnabled(ctx context.Context) error {
	if s.cfg != nil && !s.cfg.Ops.Enabled {
		return ErrOpsDisabled
	}
	if s.settingRepo == nil {
		return nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpsMonitoringEnabled)
	if err == nil {
		if isFalseSettingValue(value) {
			return ErrOpsDisabled
		}
		return nil
	}
	if errors.Is(err, ErrSettingNotFound) {
		return nil
	}
	return err
}

func normalizeUptimeQuery(input UptimeChartQuery, now time.Time) (UptimeChartQuery, error) {
	window, duration, bucketSeconds, err := parseUptimeWindow(input.Window)
	if err != nil {
		return UptimeChartQuery{}, err
	}
	dimension, err := parseUptimeDimension(input.Scope, input.Dimension)
	if err != nil {
		return UptimeChartQuery{}, err
	}
	if input.Scope == UptimeScopeUser && (input.UserID == nil || *input.UserID <= 0) {
		return UptimeChartQuery{}, infraerrors.BadRequest("UPTIME_USER_REQUIRED", "user_id is required")
	}
	return UptimeChartQuery{
		Scope:         input.Scope,
		Dimension:     dimension,
		UserID:        input.UserID,
		StartTime:     now.Add(-duration),
		EndTime:       now,
		Window:        window,
		BucketSeconds: bucketSeconds,
	}, nil
}

func parseUptimeWindow(raw string) (string, time.Duration, int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "1h":
		return "1h", time.Hour, 60, nil
	case "6h":
		return "6h", 6 * time.Hour, 300, nil
	default:
		return "", 0, 0, infraerrors.BadRequest("INVALID_UPTIME_WINDOW", "window must be 1h or 6h")
	}
}

func parseUptimeDimension(scope UptimeScope, raw UptimeDimension) (UptimeDimension, error) {
	dimension := UptimeDimension(strings.ToLower(strings.TrimSpace(string(raw))))
	if dimension == "" {
		dimension = UptimeDimensionAll
	}
	if scope == UptimeScopeUser && isUserUptimeDimension(dimension) {
		return dimension, nil
	}
	if scope == UptimeScopeAdmin && isAdminUptimeDimension(dimension) {
		return dimension, nil
	}
	return "", infraerrors.BadRequest("INVALID_UPTIME_DIMENSION", "dimension is not supported for this scope")
}

func isUserUptimeDimension(dimension UptimeDimension) bool {
	return dimension == UptimeDimensionAll ||
		dimension == UptimeDimensionAPIKey ||
		dimension == UptimeDimensionModel
}

func isAdminUptimeDimension(dimension UptimeDimension) bool {
	return dimension == UptimeDimensionAll ||
		dimension == UptimeDimensionModel ||
		dimension == UptimeDimensionGroup
}

package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type uptimeChartRepoStub struct {
	query UptimeChartQuery
}

func (s *uptimeChartRepoStub) GetUptimeBucketRows(_ context.Context, query UptimeChartQuery) ([]UptimeBucketRow, error) {
	s.query = query
	bucket := alignUptimeBucketStart(query.StartTime, query.BucketSeconds)
	return []UptimeBucketRow{
		{Bucket: bucket, Key: "gpt-5", Label: "gpt-5", Success: 3, Failure: 1},
	}, nil
}

func TestUptimeChartServiceGetChartBuildsUserModelResponse(t *testing.T) {
	repo := &uptimeChartRepoStub{}
	svc := NewUptimeChartService(repo, nil, nil)
	userID := int64(42)

	got, err := svc.GetChart(context.Background(), UptimeChartQuery{
		Scope:     UptimeScopeUser,
		UserID:    &userID,
		Window:    "1h",
		Dimension: UptimeDimensionModel,
	})

	require.NoError(t, err)
	require.Equal(t, UptimeScopeUser, repo.query.Scope)
	require.Equal(t, UptimeDimensionModel, repo.query.Dimension)
	require.Equal(t, 60, repo.query.BucketSeconds)
	require.Equal(t, int64(4), got.Summary.Total)
	require.Equal(t, int64(3), got.Summary.Success)
	require.Equal(t, int64(1), got.Summary.Failure)
	require.NotNil(t, got.Summary.Availability)
	require.Equal(t, 0.75, *got.Summary.Availability)
	require.Len(t, got.Series, 1)
	require.Equal(t, "usage_logs", got.SuccessSource)
	require.Equal(t, "ops_error_logs", got.FailureSource)
	require.True(t, got.SLAExcludesBusiness)
}

func TestUptimeChartServiceRejectsUnsupportedUserDimension(t *testing.T) {
	repo := &uptimeChartRepoStub{}
	svc := NewUptimeChartService(repo, nil, nil)
	userID := int64(42)

	_, err := svc.GetChart(context.Background(), UptimeChartQuery{
		Scope:     UptimeScopeUser,
		UserID:    &userID,
		Window:    "1h",
		Dimension: UptimeDimensionGroup,
	})

	require.Error(t, err)
	require.Equal(t, "INVALID_UPTIME_DIMENSION", infraerrors.Reason(err))
}

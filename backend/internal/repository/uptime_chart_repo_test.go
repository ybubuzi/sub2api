package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUptimeChartRepositoryGetRowsUsesUsageAndOpsErrorSources(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	start := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	userID := int64(42)
	resultRows := sqlmock.NewRows([]string{
		"bucket",
		"series_key",
		"series_label",
		"success_count",
		"failure_count",
	}).AddRow(start, "7", "Primary key", int64(2), int64(1))

	queryPattern := regexp.QuoteMeta("FROM usage_logs ul") +
		`(?s).*` + regexp.QuoteMeta("LEFT JOIN api_keys ak ON ak.id = ul.api_key_id") +
		`(?s).*` + regexp.QuoteMeta("FROM ops_error_logs o") +
		`(?s).*` + regexp.QuoteMeta("COALESCE(o.status_code, 0) >= 400") +
		`(?s).*` + regexp.QuoteMeta("o.is_count_tokens = FALSE") +
		`(?s).*` + regexp.QuoteMeta("NOT o.is_business_limited") +
		`(?s).*` + regexp.QuoteMeta("o.user_id = $3")

	mock.ExpectQuery(queryPattern).
		WithArgs(start, end, userID).
		WillReturnRows(resultRows)

	repo := NewUptimeChartRepository(db)
	got, err := repo.GetUptimeBucketRows(context.Background(), service.UptimeChartQuery{
		Scope:         service.UptimeScopeUser,
		Dimension:     service.UptimeDimensionAPIKey,
		UserID:        &userID,
		StartTime:     start,
		EndTime:       end,
		BucketSeconds: 60,
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Len(t, got, 1)
	require.Equal(t, start, got[0].Bucket)
	require.Equal(t, "7", got[0].Key)
	require.Equal(t, "Primary key", got[0].Label)
	require.Equal(t, int64(2), got[0].Success)
	require.Equal(t, int64(1), got[0].Failure)
}

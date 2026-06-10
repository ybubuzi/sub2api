package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type uptimeChartRepository struct {
	db *sql.DB
}

type uptimeDimensionSQL struct {
	successJoin  string
	errorJoin    string
	successKey   string
	successLabel string
	errorKey     string
	errorLabel   string
}

func NewUptimeChartRepository(db *sql.DB) service.UptimeChartRepository {
	return &uptimeChartRepository{db: db}
}

func (r *uptimeChartRepository) GetUptimeBucketRows(ctx context.Context, query service.UptimeChartQuery) ([]service.UptimeBucketRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil uptime chart repository")
	}
	spec := uptimeDimensionSpec(query.Dimension)
	sqlText := buildUptimeChartSQL(query, spec)
	args := uptimeChartArgs(query)
	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanUptimeBucketRows(rows)
}

func buildUptimeChartSQL(query service.UptimeChartQuery, spec uptimeDimensionSQL) string {
	successWhere := uptimeSuccessWhere(query)
	errorWhere := uptimeErrorWhere(query)
	return fmt.Sprintf(`
WITH source_rows AS (
  SELECT %s AS bucket, %s AS series_key, %s AS series_label,
         COUNT(*)::BIGINT AS success_count, 0::BIGINT AS failure_count
  FROM usage_logs ul
  %s
  %s
  GROUP BY 1, 2, 3
  UNION ALL
  SELECT %s AS bucket, %s AS series_key, %s AS series_label,
         0::BIGINT AS success_count, COUNT(*)::BIGINT AS failure_count
  FROM ops_error_logs o
  %s
  %s
  GROUP BY 1, 2, 3
)
SELECT bucket, series_key, series_label, SUM(success_count), SUM(failure_count)
FROM source_rows
GROUP BY 1, 2, 3
ORDER BY bucket ASC, SUM(success_count + failure_count) DESC, series_label ASC, series_key ASC`,
		uptimeBucketExpr("ul.created_at", query.BucketSeconds),
		spec.successKey,
		spec.successLabel,
		spec.successJoin,
		successWhere,
		uptimeBucketExpr("o.created_at", query.BucketSeconds),
		spec.errorKey,
		spec.errorLabel,
		spec.errorJoin,
		errorWhere,
	)
}

func uptimeChartArgs(query service.UptimeChartQuery) []any {
	args := []any{query.StartTime.UTC(), query.EndTime.UTC()}
	if query.Scope == service.UptimeScopeUser && query.UserID != nil {
		args = append(args, *query.UserID)
	}
	return args
}

func uptimeSuccessWhere(query service.UptimeChartQuery) string {
	clauses := []string{"ul.created_at >= $1", "ul.created_at < $2"}
	if query.Scope == service.UptimeScopeUser {
		clauses = append(clauses, "ul.user_id = $3")
	}
	return "WHERE " + strings.Join(clauses, " AND ")
}

func uptimeErrorWhere(query service.UptimeChartQuery) string {
	clauses := []string{
		"o.created_at >= $1",
		"o.created_at < $2",
		"COALESCE(o.status_code, 0) >= 400",
		"o.is_count_tokens = FALSE",
		"NOT o.is_business_limited",
	}
	if query.Scope == service.UptimeScopeUser {
		clauses = append(clauses, "o.user_id = $3")
	}
	return "WHERE " + strings.Join(clauses, " AND ")
}

func uptimeDimensionSpec(dimension service.UptimeDimension) uptimeDimensionSQL {
	switch dimension {
	case service.UptimeDimensionAPIKey:
		return uptimeAPIKeyDimensionSpec()
	case service.UptimeDimensionModel:
		return uptimeModelDimensionSpec()
	case service.UptimeDimensionGroup:
		return uptimeGroupDimensionSpec()
	default:
		return uptimeAllDimensionSpec()
	}
}

func uptimeAllDimensionSpec() uptimeDimensionSQL {
	return uptimeDimensionSQL{
		successKey: "'all'", successLabel: "'All traffic'",
		errorKey: "'all'", errorLabel: "'All traffic'",
	}
}

func uptimeAPIKeyDimensionSpec() uptimeDimensionSQL {
	return uptimeDimensionSQL{
		successJoin:  "LEFT JOIN api_keys ak ON ak.id = ul.api_key_id",
		errorJoin:    "LEFT JOIN api_keys ak ON ak.id = o.api_key_id",
		successKey:   "COALESCE(ul.api_key_id::TEXT, 'unknown')",
		errorKey:     "COALESCE(o.api_key_id::TEXT, 'unknown')",
		successLabel: "COALESCE(NULLIF(ak.name, ''), ul.api_key_id::TEXT, 'Unknown API key')",
		errorLabel:   "COALESCE(NULLIF(ak.name, ''), o.api_key_id::TEXT, 'Unknown API key')",
	}
}

func uptimeModelDimensionSpec() uptimeDimensionSQL {
	usageModel := "COALESCE(NULLIF(TRIM(ul.requested_model), ''), NULLIF(TRIM(ul.model), ''), 'unknown')"
	errorModel := "COALESCE(NULLIF(TRIM(o.requested_model), ''), NULLIF(TRIM(o.model), ''), 'unknown')"
	return uptimeDimensionSQL{
		successKey: usageModel, successLabel: usageModel,
		errorKey: errorModel, errorLabel: errorModel,
	}
}

func uptimeGroupDimensionSpec() uptimeDimensionSQL {
	groupLabel := "CASE WHEN %s.group_id IS NULL THEN 'Ungrouped' ELSE COALESCE(NULLIF(g.name, ''), %s.group_id::TEXT) END"
	return uptimeDimensionSQL{
		successJoin:  "LEFT JOIN groups g ON g.id = ul.group_id",
		errorJoin:    "LEFT JOIN groups g ON g.id = o.group_id",
		successKey:   "COALESCE(ul.group_id::TEXT, 'ungrouped')",
		errorKey:     "COALESCE(o.group_id::TEXT, 'ungrouped')",
		successLabel: fmt.Sprintf(groupLabel, "ul", "ul"),
		errorLabel:   fmt.Sprintf(groupLabel, "o", "o"),
	}
}

func uptimeBucketExpr(column string, seconds int) string {
	if seconds <= 0 {
		seconds = 60
	}
	return fmt.Sprintf(
		"to_timestamp(floor(extract(epoch from %s) / %d) * %d)",
		column,
		seconds,
		seconds,
	)
}

func scanUptimeBucketRows(rows *sql.Rows) ([]service.UptimeBucketRow, error) {
	out := make([]service.UptimeBucketRow, 0, 64)
	for rows.Next() {
		row, err := scanUptimeBucketRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanUptimeBucketRow(rows *sql.Rows) (service.UptimeBucketRow, error) {
	var row service.UptimeBucketRow
	var key, label sql.NullString
	if err := rows.Scan(&row.Bucket, &key, &label, &row.Success, &row.Failure); err != nil {
		return service.UptimeBucketRow{}, err
	}
	row.Bucket = row.Bucket.UTC()
	row.Key = uptimeSQLString(key, "unknown")
	row.Label = uptimeSQLString(label, row.Key)
	return row, nil
}

func uptimeSQLString(value sql.NullString, fallback string) string {
	if value.Valid && strings.TrimSpace(value.String) != "" {
		return value.String
	}
	return fallback
}

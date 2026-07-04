package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type relayBalanceRepository struct {
	db *sql.DB
}

func NewRelayBalanceRepository(db *sql.DB) service.RelayBalanceRepository {
	return &relayBalanceRepository{db: db}
}

func (r *relayBalanceRepository) CreateStation(ctx context.Context, station *service.RelayBalanceStation) error {
	query := `
INSERT INTO relay_balance_stations (name, base_url, script, package_json, cron_expression, enabled, next_run_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, station.Name, station.BaseURL, station.Script, station.PackageJSON, station.CronExpression, station.Enabled, station.NextRunAt, relayBalanceNullInt64(station.CreatedBy)).Scan(&station.ID, &station.CreatedAt, &station.UpdatedAt)
}

func (r *relayBalanceRepository) GetStation(ctx context.Context, id int64) (*service.RelayBalanceStation, error) {
	query := stationSelectSQL() + ` WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanRelayBalanceStation(row)
}

func (r *relayBalanceRepository) UpdateStation(ctx context.Context, station *service.RelayBalanceStation) error {
	query := `
UPDATE relay_balance_stations
SET name = $2, base_url = $3, script = $4, package_json = $5, cron_expression = $6, enabled = $7, next_run_at = $8, updated_at = NOW()
WHERE id = $1
RETURNING created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, station.ID, station.Name, station.BaseURL, station.Script, station.PackageJSON, station.CronExpression, station.Enabled, station.NextRunAt).Scan(&station.CreatedAt, &station.UpdatedAt)
}

func (r *relayBalanceRepository) DeleteStation(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM relay_balance_stations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *relayBalanceRepository) ListStations(ctx context.Context, params service.RelayBalanceListParams) ([]*service.RelayBalanceStation, int64, error) {
	where, args := relayBalanceStationWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM relay_balance_stations`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)
	query := stationSelectSQL() + where + fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := []*service.RelayBalanceStation{}
	for rows.Next() {
		station, err := scanRelayBalanceStationRows(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, station)
	}
	return items, total, rows.Err()
}

func (r *relayBalanceRepository) ListEnabledStations(ctx context.Context) ([]*service.RelayBalanceStation, error) {
	rows, err := r.db.QueryContext(ctx, stationSelectSQL()+` WHERE enabled = TRUE ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []*service.RelayBalanceStation{}
	for rows.Next() {
		station, err := scanRelayBalanceStationRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, station)
	}
	return items, rows.Err()
}

func (r *relayBalanceRepository) InsertRun(ctx context.Context, run *service.RelayBalanceRun) error {
	raw := strings.TrimSpace(run.Raw)
	if raw == "" {
		raw = "null"
	}
	query := `
INSERT INTO relay_balance_runs (station_id, station_name, balance, currency, status, stdout, stderr, error, raw, duration_ms, started_at, finished_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12)
RETURNING id`
	return r.db.QueryRowContext(ctx, query, run.StationID, run.StationName, relayBalanceNullableFloat(run.Balance), relayBalanceNullString(run.Currency), run.Status, relayBalanceTruncateText(run.Stdout), relayBalanceTruncateText(run.Stderr), relayBalanceNullString(run.Error), raw, run.DurationMs, run.StartedAt, run.FinishedAt).Scan(&run.ID)
}

func (r *relayBalanceRepository) ListRuns(ctx context.Context, params service.RelayBalanceRunListParams) ([]*service.RelayBalanceRun, int64, error) {
	where, args := relayBalanceRunWhere(params)
	if params.Granularity == "hour" || params.Granularity == "day" {
		return r.listRunsGrouped(ctx, params, where, args)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM relay_balance_runs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)
	sortOrder := relayBalanceSortOrder(params.SortOrder)
	query := runSelectSQL() + where + fmt.Sprintf(` ORDER BY started_at %s, id %s LIMIT $%d OFFSET $%d`, sortOrder, sortOrder, len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := []*service.RelayBalanceRun{}
	for rows.Next() {
		run, err := scanRelayBalanceRunRows(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, run)
	}
	return items, total, rows.Err()
}

func (r *relayBalanceRepository) listRunsGrouped(ctx context.Context, params service.RelayBalanceRunListParams, where string, args []any) ([]*service.RelayBalanceRun, int64, error) {
	grain := params.Granularity
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM (
		SELECT 1 FROM relay_balance_runs%s
		GROUP BY station_id, station_name, date_trunc('%s', started_at)
	) t`, where, grain)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)
	sortOrder := relayBalanceSortOrder(params.SortOrder)
	query := fmt.Sprintf(`
SELECT 0 AS id,
       station_id,
       station_name,
       AVG(balance) FILTER (WHERE balance IS NOT NULL) AS balance,
       COALESCE((array_remove(array_agg(currency ORDER BY started_at DESC), NULL))[1], '') AS currency,
       (array_agg(status ORDER BY started_at DESC))[1] AS status,
       '' AS stdout,
       '' AS stderr,
       format('success=%%s failed=%%s', COUNT(*) FILTER (WHERE status = 'success'), COUNT(*) FILTER (WHERE status <> 'success')) AS error,
       'null' AS raw,
       COALESCE(AVG(duration_ms)::int, 0) AS duration_ms,
       date_trunc('%s', started_at) AS started_at,
       MAX(finished_at) AS finished_at
FROM relay_balance_runs%s
GROUP BY station_id, station_name, date_trunc('%s', started_at)
ORDER BY started_at %s, station_id %s
LIMIT $%d OFFSET $%d`, grain, where, grain, sortOrder, sortOrder, len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := []*service.RelayBalanceRun{}
	for rows.Next() {
		run, err := scanRelayBalanceRunRows(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, run)
	}
	return items, total, rows.Err()
}

func (r *relayBalanceRepository) UpdateStationAfterRun(ctx context.Context, stationID int64, result service.RelayBalanceRun, nextRunAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE relay_balance_stations
SET last_balance = $2, last_currency = $3, last_status = $4, last_error = $5, last_run_at = $6, next_run_at = $7, updated_at = NOW()
WHERE id = $1`, stationID, relayBalanceNullableFloat(result.Balance), relayBalanceNullString(result.Currency), result.Status, relayBalanceNullString(result.Error), result.FinishedAt, nextRunAt)
	return err
}

func stationSelectSQL() string {
	return `SELECT id, name, base_url, script, package_json, cron_expression, enabled, last_balance, last_currency, last_status, last_error, last_run_at, next_run_at, created_by, created_at, updated_at FROM relay_balance_stations`
}

func runSelectSQL() string {
	return `SELECT id, station_id, station_name, balance, currency, status, stdout, stderr, error, COALESCE(raw::text, 'null'), duration_ms, started_at, finished_at FROM relay_balance_runs`
}

func relayBalanceStationWhere(params service.RelayBalanceListParams) (string, []any) {
	clauses := []string{}
	args := []any{}
	if params.Enabled != nil {
		args = append(args, *params.Enabled)
		clauses = append(clauses, fmt.Sprintf("enabled = $%d", len(args)))
	}
	if search := strings.TrimSpace(params.Search); search != "" {
		args = append(args, "%"+search+"%")
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR base_url ILIKE $%d)", len(args), len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func relayBalanceRunWhere(params service.RelayBalanceRunListParams) (string, []any) {
	clauses := []string{}
	args := []any{}
	if params.StationID > 0 {
		args = append(args, params.StationID)
		clauses = append(clauses, fmt.Sprintf("station_id = $%d", len(args)))
	}
	if status := strings.TrimSpace(params.Status); status != "" {
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if params.StartedFrom != nil {
		args = append(args, *params.StartedFrom)
		clauses = append(clauses, fmt.Sprintf("started_at >= $%d", len(args)))
	}
	if params.StartedTo != nil {
		args = append(args, *params.StartedTo)
		clauses = append(clauses, fmt.Sprintf("started_at <= $%d", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func relayBalanceSortOrder(sortOrder string) string {
	if strings.EqualFold(sortOrder, "asc") {
		return "ASC"
	}
	return "DESC"
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRelayBalanceStation(row rowScanner) (*service.RelayBalanceStation, error) {
	station := &service.RelayBalanceStation{}
	var lastBalance sql.NullFloat64
	var lastCurrency, lastStatus, lastError sql.NullString
	var lastRunAt, nextRunAt sql.NullTime
	var createdBy sql.NullInt64
	if err := row.Scan(&station.ID, &station.Name, &station.BaseURL, &station.Script, &station.PackageJSON, &station.CronExpression, &station.Enabled, &lastBalance, &lastCurrency, &lastStatus, &lastError, &lastRunAt, &nextRunAt, &createdBy, &station.CreatedAt, &station.UpdatedAt); err != nil {
		return nil, err
	}
	if lastBalance.Valid {
		station.LastBalance = &lastBalance.Float64
	}
	station.LastCurrency = lastCurrency.String
	station.LastStatus = lastStatus.String
	station.LastError = lastError.String
	if lastRunAt.Valid {
		station.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		station.NextRunAt = &nextRunAt.Time
	}
	if createdBy.Valid {
		station.CreatedBy = createdBy.Int64
	}
	return station, nil
}

func scanRelayBalanceStationRows(rows *sql.Rows) (*service.RelayBalanceStation, error) {
	return scanRelayBalanceStation(rows)
}

func scanRelayBalanceRunRows(rows *sql.Rows) (*service.RelayBalanceRun, error) {
	run := &service.RelayBalanceRun{}
	var balance sql.NullFloat64
	var currency, stdout, stderr, errText sql.NullString
	var finishedAt sql.NullTime
	if err := rows.Scan(&run.ID, &run.StationID, &run.StationName, &balance, &currency, &run.Status, &stdout, &stderr, &errText, &run.Raw, &run.DurationMs, &run.StartedAt, &finishedAt); err != nil {
		return nil, err
	}
	if balance.Valid {
		run.Balance = &balance.Float64
	}
	run.Currency = currency.String
	run.Stdout = stdout.String
	run.Stderr = stderr.String
	run.Error = errText.String
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	return run, nil
}

func relayBalanceNullableFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func relayBalanceNullString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func relayBalanceNullInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func relayBalanceTruncateText(v string) string {
	if len(v) > 64000 {
		return v[:64000]
	}
	return v
}

func (r *relayBalanceRepository) GetTotalBalance(ctx context.Context) (*service.RelayBalanceTotalResponse, error) {
	var totalBalance float64
	var stationCount int
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(last_balance), 0), COUNT(*) FILTER (WHERE last_status = 'success')
		FROM relay_balance_stations
		WHERE last_balance IS NOT NULL AND last_status = 'success'
	`).Scan(&totalBalance, &stationCount)
	if err != nil {
		return nil, fmt.Errorf("get total balance: %w", err)
	}
	return &service.RelayBalanceTotalResponse{
		TotalBalance: totalBalance,
		Currency:     "USD",
		StationCount: stationCount,
	}, nil
}

func (r *relayBalanceRepository) GetTrend(ctx context.Context, params service.RelayBalanceTrendParams) ([]service.RelayBalanceTrendPoint, error) {
	query := `
		SELECT 
			date_trunc($1, started_at) AS bucket,
			station_id,
			station_name,
			balance
		FROM relay_balance_runs
		WHERE started_at >= $2 AND started_at <= $3 AND balance IS NOT NULL
		ORDER BY station_id, bucket DESC
	`

	rows, err := r.db.QueryContext(ctx, query, params.Granularity, *params.StartedFrom, *params.StartedTo)
	if err != nil {
		return nil, fmt.Errorf("query trend: %w", err)
	}
	defer func() { _ = rows.Close() }()

	pointMap := make(map[string]service.RelayBalanceTrendPoint)

	for rows.Next() {
		var bucket time.Time
		var stationID int64
		var stationName string
		var balance float64

		if err := rows.Scan(&bucket, &stationID, &stationName, &balance); err != nil {
			return nil, fmt.Errorf("scan trend row: %w", err)
		}

		key := fmt.Sprintf("%d_%s", stationID, bucket.Format(time.RFC3339))
		if _, exists := pointMap[key]; !exists {
			pointMap[key] = service.RelayBalanceTrendPoint{
				Bucket:      bucket,
				StationID:   stationID,
				StationName: stationName,
				Balance:     balance,
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	points := make([]service.RelayBalanceTrendPoint, 0, len(pointMap))
	for _, point := range pointMap {
		points = append(points, point)
	}

	return points, nil
}

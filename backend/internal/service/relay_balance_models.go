package service

import "time"

type RelayBalanceStation struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	BaseURL        string     `json:"base_url"`
	Script         string     `json:"script"`
	PackageJSON    string     `json:"package_json"`
	CronExpression string     `json:"cron_expression"`
	Enabled        bool       `json:"enabled"`
	LastBalance    *float64   `json:"last_balance"`
	LastCurrency   string     `json:"last_currency"`
	LastStatus     string     `json:"last_status"`
	LastError      string     `json:"last_error"`
	LastRunAt      *time.Time `json:"last_run_at"`
	NextRunAt      *time.Time `json:"next_run_at"`
	CreatedBy      int64      `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type RelayBalanceRun struct {
	ID          int64      `json:"id"`
	StationID   int64      `json:"station_id"`
	StationName string     `json:"station_name"`
	Balance     *float64   `json:"balance"`
	Currency    string     `json:"currency"`
	Status      string     `json:"status"`
	Stdout      string     `json:"stdout"`
	Stderr      string     `json:"stderr"`
	Error       string     `json:"error"`
	Raw         string     `json:"raw"`
	DurationMs  int        `json:"duration_ms"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
}

type RelayBalanceListParams struct {
	Page     int
	PageSize int
	Search   string
	Enabled  *bool
}

type RelayBalanceRunListParams struct {
	Page        int
	PageSize    int
	StationID   int64
	Status      string
	StartedFrom *time.Time
	StartedTo   *time.Time
	SortOrder   string
	Granularity string
}

type RelayBalanceStationInput struct {
	Name           string
	BaseURL        string
	Script         string
	PackageJSON    string
	CronExpression string
	Enabled        bool
	CreatedBy      int64
}

type RelayBalanceTrendParams struct {
	StartedFrom *time.Time
	StartedTo   *time.Time
	Granularity string
}

type RelayBalanceTrendPoint struct {
	Bucket      time.Time `json:"bucket"`
	StationID   int64     `json:"station_id"`
	StationName string    `json:"station_name"`
	Balance     float64   `json:"balance"`
}

type RelayBalanceTrendResponse struct {
	Buckets []string                  `json:"buckets"`
	Series  []RelayBalanceTrendSeries `json:"series"`
	Total   []float64                 `json:"total"`
}

type RelayBalanceTrendSeries struct {
	StationID   int64     `json:"station_id"`
	StationName string    `json:"station_name"`
	Balances    []float64 `json:"balances"`
}

type RelayBalanceTotalResponse struct {
	TotalBalance float64 `json:"total_balance"`
	Currency     string  `json:"currency"`
	StationCount int     `json:"station_count"`
}

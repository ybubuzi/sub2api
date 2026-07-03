package service

import (
	"context"
	"time"
)

type RelayBalanceRepository interface {
	CreateStation(ctx context.Context, station *RelayBalanceStation) error
	GetStation(ctx context.Context, id int64) (*RelayBalanceStation, error)
	UpdateStation(ctx context.Context, station *RelayBalanceStation) error
	DeleteStation(ctx context.Context, id int64) error
	ListStations(ctx context.Context, params RelayBalanceListParams) ([]*RelayBalanceStation, int64, error)
	ListEnabledStations(ctx context.Context) ([]*RelayBalanceStation, error)
	UpdateStationAfterRun(ctx context.Context, stationID int64, result RelayBalanceRun, nextRunAt *time.Time) error
	ListRuns(ctx context.Context, params RelayBalanceRunListParams) ([]*RelayBalanceRun, int64, error)
	InsertRun(ctx context.Context, run *RelayBalanceRun) error
	GetTrend(ctx context.Context, params RelayBalanceTrendParams) ([]RelayBalanceTrendPoint, error)
}

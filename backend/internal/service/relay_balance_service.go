package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

var relayBalanceCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

type RelayBalanceService struct {
	repo     RelayBalanceRepository
	executor RelayBalanceExecutor
	cron     *cron.Cron
	mu       sync.Mutex
	entries  map[int64]cron.EntryID
}

func NewRelayBalanceService(repo RelayBalanceRepository, executor RelayBalanceExecutor) *RelayBalanceService {
	return &RelayBalanceService{repo: repo, executor: executor, entries: make(map[int64]cron.EntryID)}
}

func ProvideRelayBalanceService(repo RelayBalanceRepository, executor RelayBalanceExecutor) *RelayBalanceService {
	svc := NewRelayBalanceService(repo, executor)
	svc.Start()
	return svc
}

func (s *RelayBalanceService) Start() {
	s.mu.Lock()
	if s.cron != nil {
		s.mu.Unlock()
		return
	}
	s.cron = cron.New(cron.WithParser(relayBalanceCronParser), cron.WithLocation(time.Local))
	s.cron.Start()
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stations, err := s.repo.ListEnabledStations(ctx)
	if err != nil {
		log.Printf("[RelayBalance] load enabled stations failed: %v", err)
		return
	}
	for _, station := range stations {
		if err := s.Schedule(station); err != nil {
			log.Printf("[RelayBalance] schedule station %d failed: %v", station.ID, err)
		}
	}
}

func (s *RelayBalanceService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron != nil {
		ctx := s.cron.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(3 * time.Second):
		}
		s.cron = nil
	}
	s.entries = make(map[int64]cron.EntryID)
}

func (s *RelayBalanceService) ListStations(ctx context.Context, params RelayBalanceListParams) ([]*RelayBalanceStation, int64, error) {
	normalizeRelayBalancePagination(&params.Page, &params.PageSize)
	return s.repo.ListStations(ctx, params)
}

func (s *RelayBalanceService) GetStation(ctx context.Context, id int64) (*RelayBalanceStation, error) {
	return s.repo.GetStation(ctx, id)
}

func (s *RelayBalanceService) CreateStation(ctx context.Context, in RelayBalanceStationInput) (*RelayBalanceStation, error) {
	station, err := s.stationFromInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateStation(ctx, station); err != nil {
		return nil, err
	}
	if station.Enabled {
		_ = s.Schedule(station)
	}
	return station, nil
}

func (s *RelayBalanceService) UpdateStation(ctx context.Context, id int64, in RelayBalanceStationInput) (*RelayBalanceStation, error) {
	station, err := s.stationFromInput(in)
	if err != nil {
		return nil, err
	}
	station.ID = id
	if err := s.repo.UpdateStation(ctx, station); err != nil {
		return nil, err
	}
	s.Unschedule(id)
	if station.Enabled {
		_ = s.Schedule(station)
	}
	return station, nil
}

func (s *RelayBalanceService) DeleteStation(ctx context.Context, id int64) error {
	s.Unschedule(id)
	return s.repo.DeleteStation(ctx, id)
}

func (s *RelayBalanceService) ListRuns(ctx context.Context, params RelayBalanceRunListParams) ([]*RelayBalanceRun, int64, error) {
	normalizeRelayBalancePagination(&params.Page, &params.PageSize)
	params.SortOrder = strings.ToLower(strings.TrimSpace(params.SortOrder))
	if params.SortOrder != "asc" {
		params.SortOrder = "desc"
	}
	params.Granularity = strings.ToLower(strings.TrimSpace(params.Granularity))
	if params.Granularity != "hour" && params.Granularity != "day" {
		params.Granularity = ""
	}
	return s.repo.ListRuns(ctx, params)
}

func (s *RelayBalanceService) RunNow(ctx context.Context, id int64) (*RelayBalanceRun, error) {
	station, err := s.repo.GetStation(ctx, id)
	if err != nil {
		return nil, err
	}
	run := s.executeStation(ctx, station)
	return &run, nil
}

func (s *RelayBalanceService) Schedule(station *RelayBalanceStation) error {
	if station == nil || !station.Enabled {
		return nil
	}
	if _, err := relayBalanceCronParser.Parse(station.CronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron == nil {
		return nil
	}
	if old, ok := s.entries[station.ID]; ok {
		s.cron.Remove(old)
		delete(s.entries, station.ID)
	}
	entryID, err := s.cron.AddFunc(station.CronExpression, func() {
		ctx, cancel := context.WithTimeout(context.Background(), relayBalanceExecTimeout+90*time.Second)
		defer cancel()
		latest, err := s.repo.GetStation(ctx, station.ID)
		if err != nil {
			log.Printf("[RelayBalance] load station %d failed: %v", station.ID, err)
			return
		}
		if !latest.Enabled {
			return
		}
		s.executeStation(ctx, latest)
	})
	if err != nil {
		return err
	}
	s.entries[station.ID] = entryID
	return nil
}

func (s *RelayBalanceService) Unschedule(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron == nil {
		return
	}
	if entryID, ok := s.entries[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
}

func (s *RelayBalanceService) executeStation(ctx context.Context, station *RelayBalanceStation) RelayBalanceRun {
	run := s.executor.Execute(ctx, station)
	if err := s.repo.InsertRun(ctx, &run); err != nil {
		log.Printf("[RelayBalance] insert run station=%d failed: %v", station.ID, err)
	}
	next := nextRelayBalanceRun(station.CronExpression, time.Now())
	if err := s.repo.UpdateStationAfterRun(ctx, station.ID, run, next); err != nil {
		log.Printf("[RelayBalance] update station after run station=%d failed: %v", station.ID, err)
	}
	return run
}

func (s *RelayBalanceService) stationFromInput(in RelayBalanceStationInput) (*RelayBalanceStation, error) {
	name := strings.TrimSpace(in.Name)
	baseURL := strings.TrimSpace(in.BaseURL)
	script := strings.TrimSpace(in.Script)
	cronExpr := strings.TrimSpace(in.CronExpression)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	if script == "" {
		return nil, fmt.Errorf("script is required")
	}
	if cronExpr == "" {
		cronExpr = "0 * * * *"
	}
	if _, err := relayBalanceCronParser.Parse(cronExpr); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	pkg := normalizePackageJSON(in.PackageJSON)
	if !json.Valid([]byte(pkg)) {
		return nil, fmt.Errorf("package_json must be valid JSON")
	}
	return &RelayBalanceStation{Name: name, BaseURL: baseURL, Script: script, PackageJSON: pkg, CronExpression: cronExpr, Enabled: in.Enabled, CreatedBy: in.CreatedBy, NextRunAt: nextRelayBalanceRun(cronExpr, time.Now())}, nil
}

func normalizeRelayBalancePagination(page, pageSize *int) {
	if *page < 1 {
		*page = 1
	}
	if *pageSize < 1 || *pageSize > 200 {
		*pageSize = 20
	}
}

func nextRelayBalanceRun(expr string, from time.Time) *time.Time {
	sched, err := relayBalanceCronParser.Parse(expr)
	if err != nil {
		return nil
	}
	next := sched.Next(from)
	return &next
}

func (s *RelayBalanceService) GetTrend(ctx context.Context, params RelayBalanceTrendParams) (*RelayBalanceTrendResponse, error) {
	if params.Granularity != "hour" && params.Granularity != "day" {
		return nil, fmt.Errorf("granularity must be 'hour' or 'day'")
	}
	if params.StartedFrom == nil || params.StartedTo == nil {
		return nil, fmt.Errorf("started_from and started_to are required")
	}
	if params.StartedFrom.After(*params.StartedTo) {
		return nil, fmt.Errorf("started_from must be before started_to")
	}

	points, err := s.repo.GetTrend(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get trend: %w", err)
	}

	bucketMap := make(map[time.Time]bool)
	stationMap := make(map[int64]string)
	bucketStationBalance := make(map[time.Time]map[int64]float64)

	for _, point := range points {
		bucketMap[point.Bucket] = true
		stationMap[point.StationID] = point.StationName

		if _, ok := bucketStationBalance[point.Bucket]; !ok {
			bucketStationBalance[point.Bucket] = make(map[int64]float64)
		}
		bucketStationBalance[point.Bucket][point.StationID] = point.Balance
	}

	buckets := make([]time.Time, 0, len(bucketMap))
	for bucket := range bucketMap {
		buckets = append(buckets, bucket)
	}
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Before(buckets[j])
	})

	response := &RelayBalanceTrendResponse{
		Buckets: make([]string, len(buckets)),
		Series:  make([]RelayBalanceTrendSeries, 0),
		Total:   make([]float64, len(buckets)),
	}

	for i, bucket := range buckets {
		response.Buckets[i] = bucket.Format(time.RFC3339)
	}

	for stationID, stationName := range stationMap {
		series := RelayBalanceTrendSeries{
			StationID:   stationID,
			StationName: stationName,
			Balances:    make([]float64, len(buckets)),
		}

		for i, bucket := range buckets {
			if stationBalances, ok := bucketStationBalance[bucket]; ok {
				if balance, ok := stationBalances[stationID]; ok {
					series.Balances[i] = balance
					response.Total[i] += balance
				}
			}
		}

		response.Series = append(response.Series, series)
	}

	return response, nil
}

func (s *RelayBalanceService) GetTotalBalance(ctx context.Context) (*RelayBalanceTotalResponse, error) {
	return s.repo.GetTotalBalance(ctx)
}

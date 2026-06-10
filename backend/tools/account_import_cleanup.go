package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

const defaultImportPath = `F:\PROJECT\ZDF\GYK\GYK_PSD\temp\219_converted.json`

type appConfig struct {
	Database databaseConfig `yaml:"database"`
}

type databaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type importPayload struct {
	Accounts []importAccount `json:"accounts"`
	Data     *importData     `json:"data"`
}

type importData struct {
	Accounts []importAccount `json:"accounts"`
}

type importAccount struct {
	Name string `json:"name"`
}

type duplicateRow struct {
	Name       string
	IDs        []int64
	KeepID     int64
	DeleteIDs  []int64
	TotalCount int
}

func main() {
	path := flag.String("file", defaultImportPath, "import json file")
	configPath := flag.String("config", "config.yaml", "backend config yaml")
	apply := flag.Bool("apply", false, "soft-delete duplicate imported account copies")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	names, err := readImportNames(*path)
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := loadDBConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("postgres", dsn(cfg))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	rows, err := findDuplicates(ctx, db, names)
	if err != nil {
		log.Fatal(err)
	}
	printPreview(rows)
	if !*apply {
		fmt.Println("dry_run=true")
		return
	}
	deleted, err := softDeleteDuplicates(ctx, db, rows)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("dry_run=false soft_deleted=%d\n", deleted)
}

func readImportNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload importPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	accounts := payload.Accounts
	if len(accounts) == 0 && payload.Data != nil {
		accounts = payload.Data.Accounts
	}
	return uniqueNames(accounts), nil
}

func uniqueNames(accounts []importAccount) []string {
	seen := make(map[string]struct{}, len(accounts))
	out := make([]string, 0, len(accounts))
	for _, account := range accounts {
		name := strings.TrimSpace(account.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func loadDBConfig(path string) (databaseConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return databaseConfig{}, err
	}
	var cfg appConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return databaseConfig{}, err
	}
	return applyDBEnv(cfg.Database)
}

func applyDBEnv(cfg databaseConfig) (databaseConfig, error) {
	cfg.Host = env("PGHOST", cfg.Host)
	cfg.User = env("PGUSER", cfg.User)
	cfg.Password = env("PGPASSWORD", cfg.Password)
	cfg.DBName = env("PGDATABASE", cfg.DBName)
	cfg.SSLMode = env("PGSSLMODE", cfg.SSLMode)
	if rawPort := os.Getenv("PGPORT"); rawPort != "" {
		port, err := strconv.Atoi(rawPort)
		if err != nil {
			return databaseConfig{}, err
		}
		cfg.Port = port
	}
	return cfg, nil
}

func dsn(cfg databaseConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func findDuplicates(ctx context.Context, db *sql.DB, names []string) ([]duplicateRow, error) {
	if len(names) == 0 {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, duplicateQuery(), pqArray(names))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDuplicates(rows)
}

func duplicateQuery() string {
	return `
WITH target_names AS (SELECT UNNEST($1::text[]) AS name),
ranked AS (
  SELECT a.name, a.id, ROW_NUMBER() OVER (
    PARTITION BY a.name ORDER BY a.updated_at DESC, a.created_at DESC, a.id DESC
  ) AS rn, COUNT(*) OVER (PARTITION BY a.name) AS total_count
  FROM accounts a JOIN target_names t ON t.name = a.name
  WHERE a.deleted_at IS NULL
)
SELECT name, ARRAY_AGG(id ORDER BY id), MAX(id) FILTER (WHERE rn = 1),
       ARRAY_REMOVE(ARRAY_AGG(CASE WHEN rn > 1 THEN id END ORDER BY id), NULL),
       MAX(total_count)
FROM ranked WHERE total_count > 1 GROUP BY name ORDER BY name`
}

func scanDuplicates(rows *sql.Rows) ([]duplicateRow, error) {
	out := []duplicateRow{}
	for rows.Next() {
		var row duplicateRow
		if err := rows.Scan(&row.Name, pqArray(&row.IDs), &row.KeepID, pqArray(&row.DeleteIDs), &row.TotalCount); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func printPreview(rows []duplicateRow) {
	totalDelete := 0
	for _, row := range rows {
		totalDelete += len(row.DeleteIDs)
		fmt.Printf("name=%s total=%d keep=%d delete=%v all=%v\n", row.Name, row.TotalCount, row.KeepID, row.DeleteIDs, row.IDs)
	}
	fmt.Printf("duplicate_names=%d duplicate_rows_to_soft_delete=%d\n", len(rows), totalDelete)
}

func softDeleteDuplicates(ctx context.Context, db *sql.DB, rows []duplicateRow) (int64, error) {
	ids := flattenDeleteIDs(rows)
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := db.ExecContext(ctx, "UPDATE accounts SET deleted_at = NOW(), updated_at = NOW() WHERE id = ANY($1::bigint[]) AND deleted_at IS NULL", pqArray(ids))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func flattenDeleteIDs(rows []duplicateRow) []int64 {
	out := []int64{}
	for _, row := range rows {
		out = append(out, row.DeleteIDs...)
	}
	return out
}

func pqArray(v any) any {
	return pq.Array(v)
}

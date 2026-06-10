package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	host := getenv("PGHOST", "192.168.8.211")
	port := getenv("PGPORT", "25432")
	user := getenv("PGUSER", "bubuzi")
	password := os.Getenv("PGPASSWORD")
	maintenanceDB := getenv("PGDATABASE", "catenary-db")
	targetDB := getenv("PGTARGETDB", "sub2api")

	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`).MatchString(targetDB) {
		log.Fatalf("invalid target database name: %s", targetDB)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, maintenanceDB)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect maintenance database: %v", err)
	}

	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", targetDB).Scan(&exists); err != nil {
		log.Fatalf("check target database: %v", err)
	}
	if exists {
		log.Printf("database %s already exists", targetDB)
		return
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", targetDB)); err != nil {
		log.Fatalf("create target database: %v", err)
	}
	log.Printf("database %s created", targetDB)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

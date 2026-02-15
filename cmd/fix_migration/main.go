package main

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"docflow/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	cfgPath := "config.json"
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfgPath = config.GetDefaultConfigPath()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", cfgPath, err)
	}

	// Construct URL for golang-migrate
	// postgres://user:password@host:port/dbname?sslmode=disable
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		url.QueryEscape(cfg.Database.Password),
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	m, err := migrate.New(
		"file://internal/database/migrations",
		dsn,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Force(4); err != nil {
		log.Fatalf("Failed to force version: %v", err)
	}

	fmt.Println("Successfully forced migration version to 4")
}

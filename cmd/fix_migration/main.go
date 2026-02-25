package main

import (
	"fmt"
	"log"
	"os"

	"docflow/internal/config"

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

	if err := ForceMigration(&cfg.Database, "internal/database/migrations", 4); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Successfully forced migration version to 4")
}

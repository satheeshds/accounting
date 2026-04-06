package main

import (
	"fmt"
	"log/slog"
	"os"

	_ "github.com/lib/pq"
	"github.com/satheeshds/portal/db"
)

func main() {
	// Configure structured logging
	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	// Migrate and generate occurrences for all tenants immediately on startup (gap recovery),
	// then repeat daily at midnight.
	if err := runForAllTenants(); err != nil {
		slog.Warn("migration and occurrence generation failed on startup", "error", err)
	}

}

// runForAllTenants reads configuration from env vars and calls
// db.MigrateAndGenerateAllTenants, which handles tenant discovery, credential
// rotation, DB connection, and per-tenant migration + occurrence generation.
func runForAllTenants() error {
	controlURL := os.Getenv("NEXUS_CONTROL_URL")
	if controlURL == "" {
		controlURL = "http://nexus-control:8080"
	}
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		return fmt.Errorf("ADMIN_API_KEY is required")
	}
	nexusHost := os.Getenv("NEXUS_HOST")
	if nexusHost == "" {
		nexusHost = "nexus-gateway"
	}
	nexusPort := os.Getenv("NEXUS_PORT")
	if nexusPort == "" {
		nexusPort = "5433"
	}

	return db.MigrateAndGenerateAllTenants(controlURL, adminKey, nexusHost, nexusPort)
}

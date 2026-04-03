package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/satheeshds/portal/db"
)

func main() {
	// Configure structured logging
	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	// Run schema migrations via nexus-control admin API (applies to all tenants).
	if err := db.Migrate(); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Open a database connection for occurrence generation.
	database, err := db.Open()
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Generate recurring payment occurrences immediately on startup (gap recovery),
	// then repeat daily at midnight.
	if err := db.GenerateOccurrences(database); err != nil {
		slog.Warn("occurrence generation failed on startup", "error", err)
	}

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(time.Until(next))
		if err := db.GenerateOccurrences(database); err != nil {
			slog.Warn("daily occurrence generation failed", "error", err)
		}
	}
}

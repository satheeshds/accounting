package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/lib/pq"
)

// OpenWithCredentials opens a single-connection PortalDB using the given
// tenant_id as the PostgreSQL username and the JWT token as the password.
// or service account credentials (username, password)
// It reads NEXUS_HOST (default "localhost"), NEXUS_PORT (default "5433"),
// NEXUS_DATABASE (default "lake"), and NEXUS_SCHEMA (default: same as NEXUS_DATABASE)
// from the environment.
// The connection is not pinged; the first query will surface any auth errors.
func OpenWithCredentials(tenantID, token string) (*PortalDB, error) {
	host := os.Getenv("NEXUS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("NEXUS_PORT")
	if port == "" {
		port = "5433"
	}
	database := os.Getenv("NEXUS_DATABASE")
	if database == "" {
		database = "lake"
	}
	schema := os.Getenv("NEXUS_SCHEMA")
	if schema == "" {
		schema = database
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s search_path=%s sslmode=disable",
		host, port, tenantID, token, database, schema)
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open per-request database connection: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	return WrapDB(sqlDB), nil
}

// Open creates and returns a PortalDB connection to the Nexus gateway.
// The connection DSN is read from the DATABASE_URL environment variable.
// If DATABASE_URL is not set, individual NEXUS_HOST, NEXUS_PORT, NEXUS_USER,
// NEXUS_PASSWORD, NEXUS_DATABASE, and NEXUS_SCHEMA variables are used, defaulting
// to a local Nexus instance on port 5433. NEXUS_SCHEMA defaults to NEXUS_DATABASE.
func Open() (*PortalDB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("NEXUS_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("NEXUS_PORT")
		if port == "" {
			port = "5433"
		}
		user := os.Getenv("NEXUS_USER")
		if user == "" {
			user = "portal"
		}
		password := os.Getenv("NEXUS_PASSWORD")
		database := os.Getenv("NEXUS_DATABASE")
		if database == "" {
			database = "lake"
		}
		schema := os.Getenv("NEXUS_SCHEMA")
		if schema == "" {
			schema = database
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s search_path=%s sslmode=disable",
			host, port, user, password, database, schema)
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to nexus gateway: %w", err)
	}

	slog.Info("connected to nexus gateway")
	return WrapDB(sqlDB), nil
}

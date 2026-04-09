package db

import (
	"fmt"
	"log/slog"
)

// Migration represents a single versioned schema migration with both an upgrade
// (Up) and a rollback (Down) SQL statement.
type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

// ensureMigrationsTable creates the schema_migrations tracking table if it does
// not already exist. It uses the underlying *sql.DB directly to avoid the
// lake. schema prefix injected by rebind for application tables.
func ensureMigrationsTable(pdb *PortalDB) error {
	_, err := pdb.DB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER NOT NULL,
		description TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL
	)`)
	return err
}

// appliedVersions returns the set of migration versions already present in the
// schema_migrations table.  It uses the underlying *sql.DB directly so that
// the table reference is not rewritten by rebind.
func appliedVersions(pdb *PortalDB) (map[int]bool, error) {
	rows, err := pdb.DB.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

// MigrateDB applies all pending up-migrations to the provided database in
// version order.  It creates the schema_migrations tracking table on first
// use.  This function is idempotent: already-applied migrations are skipped.
//
// This is intended for use in tests where a live Nexus control endpoint is
// not available (e.g. an in-process DuckDB instance).
func MigrateDB(db *PortalDB) error {
	slog.Info("running database migrations via db connection")

	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	applied, err := appliedVersions(db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.Version] {
			slog.Debug("migration already applied, skipping", "version", m.Version, "description", m.Description)
			continue
		}
		slog.Info("applying migration", "version", m.Version, "description", m.Description)
		if _, err := db.Exec(m.Up); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Description, err)
		}
		if _, err := db.DB.Exec(
			`INSERT INTO schema_migrations (version, description, applied_at) VALUES ($1, $2, NOW())`,
			m.Version, m.Description,
		); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}
	}

	slog.Info("database migrations complete")
	return nil
}

// RollbackDB rolls back the last n applied migrations in reverse order.
// Pass n <= 0 to roll back all applied migrations.
func RollbackDB(db *PortalDB, n int) error {
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	applied, err := appliedVersions(db)
	if err != nil {
		return err
	}

	// Build the list of applied migrations in reverse order.
	var toRollback []Migration
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		if applied[m.Version] {
			toRollback = append(toRollback, m)
		}
	}

	if n > 0 && n < len(toRollback) {
		toRollback = toRollback[:n]
	}

	for _, m := range toRollback {
		slog.Info("rolling back migration", "version", m.Version, "description", m.Description)
		if m.Down != "" {
			if _, err := db.Exec(m.Down); err != nil {
				return fmt.Errorf("rollback of migration %d (%s) failed: %w", m.Version, m.Description, err)
			}
		}
		if _, err := db.DB.Exec(`DELETE FROM schema_migrations WHERE version = $1`, m.Version); err != nil {
			return fmt.Errorf("failed to remove migration %d from schema_migrations: %w", m.Version, err)
		}
	}

	slog.Info("rollback complete", "rolled_back", len(toRollback))
	return nil
}

// MigrateTenant runs schema migrations for a single tenant database.
// Occurrence generation is handled separately by the platform service.
func MigrateTenant(tenantDB *PortalDB, tenantID string) error {
	slog.Info("migrating tenant schema", "tenant_id", tenantID)

	if err := MigrateDB(tenantDB); err != nil {
		return fmt.Errorf("migration failed for tenant %s: %w", tenantID, err)
	}

	slog.Info("migration complete for tenant", "tenant_id", tenantID)
	return nil
}

// migrations is the ordered list of all versioned schema migrations.
// Each migration must have a unique, monotonically increasing Version.
// The Up statement is applied during MigrateDB; the Down statement is
// applied during RollbackDB to undo the change.
var migrations = []Migration{
	{
		Version:     1,
		Description: "create accounts table",
		Up: `CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			opening_balance INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS accounts`,
	},
	{
		Version:     2,
		Description: "create contacts table",
		Up: `CREATE TABLE IF NOT EXISTS contacts (
			id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS contacts`,
	},
	{
		Version:     3,
		Description: "create bills table",
		Up: `CREATE TABLE IF NOT EXISTS bills (
			id INTEGER NOT NULL,
			contact_id INTEGER,
			bill_number TEXT,
			issue_date DATE,
			due_date DATE,
			amount INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'draft',
			file_url TEXT,
			notes TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS bills`,
	},
	{
		Version:     4,
		Description: "create invoices table",
		Up: `CREATE TABLE IF NOT EXISTS invoices (
			id INTEGER NOT NULL,
			contact_id INTEGER,
			invoice_number TEXT,
			issue_date DATE,
			due_date DATE,
			amount INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'draft',
			file_url TEXT,
			notes TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS invoices`,
	},
	{
		Version:     5,
		Description: "create transactions table",
		Up: `CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER NOT NULL,
			account_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			amount INTEGER NOT NULL DEFAULT 0,
			transaction_date DATE,
			description TEXT,
			reference TEXT,
			transfer_account_id INTEGER,
			contact_id INTEGER,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS transactions`,
	},
	{
		Version:     6,
		Description: "create transaction_documents table",
		// No CHECK constraint on document_type — valid types are enforced by the application layer.
		Up: `CREATE TABLE IF NOT EXISTS transaction_documents (
			id INTEGER NOT NULL,
			transaction_id INTEGER NOT NULL,
			document_type TEXT NOT NULL,
			document_id INTEGER NOT NULL,
			amount INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS transaction_documents`,
	},
	{
		Version:     7,
		Description: "create payouts table",
		Up: `CREATE TABLE IF NOT EXISTS payouts (
			id INTEGER NOT NULL,
			outlet_name TEXT NOT NULL,
			platform TEXT NOT NULL,
			period_start DATE,
			period_end DATE,
			settlement_date TEXT,
			total_orders INTEGER NOT NULL DEFAULT 0,
			gross_sales_amt INTEGER NOT NULL DEFAULT 0,
			restaurant_discount_amt INTEGER NOT NULL DEFAULT 0,
			platform_commission_amt INTEGER NOT NULL DEFAULT 0,
			taxes_tcs_tds_amt INTEGER NOT NULL DEFAULT 0,
			marketing_ads_amt INTEGER NOT NULL DEFAULT 0,
			final_payout_amt INTEGER NOT NULL DEFAULT 0,
			utr_number TEXT,
			created_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS payouts`,
	},
	{
		Version:     8,
		Description: "create recurring_payments table",
		Up: `CREATE TABLE IF NOT EXISTS recurring_payments (
			id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			amount INTEGER NOT NULL,
			account_id INTEGER NOT NULL,
			contact_id INTEGER,
			frequency TEXT NOT NULL,
			interval INTEGER NOT NULL DEFAULT 1,
			start_date DATE NOT NULL,
			end_date DATE,
			next_due_date DATE,
			last_generated_date DATE,
			status TEXT NOT NULL DEFAULT 'active',
			description TEXT,
			reference TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS recurring_payments`,
	},
	{
		Version:     9,
		Description: "create recurring_payment_occurrences table",
		// Auto-generated by the server on startup and via a daily background job.
		Up: `CREATE TABLE IF NOT EXISTS recurring_payment_occurrences (
			id INTEGER NOT NULL,
			recurring_payment_id INTEGER NOT NULL,
			due_date DATE NOT NULL,
			amount INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS recurring_payment_occurrences`,
	},
	{
		Version:     10,
		Description: "create bill_items table",
		Up: `CREATE TABLE IF NOT EXISTS bill_items (
			id INTEGER NOT NULL,
			bill_id INTEGER NOT NULL,
			description TEXT NOT NULL,
			quantity DOUBLE NOT NULL DEFAULT 1,
			unit TEXT,
			unit_price INTEGER NOT NULL DEFAULT 0,
			amount INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS bill_items`,
	},
	{
		Version:     11,
		Description: "create invoice_items table",
		Up: `CREATE TABLE IF NOT EXISTS invoice_items (
			id INTEGER NOT NULL,
			invoice_id INTEGER NOT NULL,
			description TEXT NOT NULL,
			quantity DOUBLE NOT NULL DEFAULT 1,
			unit TEXT,
			unit_price INTEGER NOT NULL DEFAULT 0,
			amount INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		Down: `DROP TABLE IF EXISTS invoice_items`,
	},
}

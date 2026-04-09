package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openTestDB opens an in-file DuckDB database suitable for migration tests.
// It returns a *PortalDB and a cleanup function.
func openTestDB(t *testing.T) *PortalDB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, fmt.Sprintf("test_migrations_%s.db", t.Name()))
	rawDB, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { rawDB.Close() })
	return WrapDB(rawDB)
}

// countApplied returns the number of rows in schema_migrations.
func countApplied(t *testing.T, db *PortalDB) int {
	t.Helper()
	var n int
	row := db.DB.QueryRow(`SELECT COUNT(*) FROM schema_migrations`)
	if err := row.Scan(&n); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	return n
}

// tableExists reports whether the named table exists in the database.
func tableExists(t *testing.T, db *PortalDB, name string) bool {
	t.Helper()
	row := db.DB.QueryRow(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = $1`, name)
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("check table existence %q: %v", name, err)
	}
	return n > 0
}

// TestMigrateDB_AppliesAllMigrations verifies that MigrateDB creates all
// expected application tables and records each migration in schema_migrations.
func TestMigrateDB_AppliesAllMigrations(t *testing.T) {
	db := openTestDB(t)

	if err := MigrateDB(db); err != nil {
		t.Fatalf("MigrateDB: %v", err)
	}

	// Every declared migration must be recorded.
	if got, want := countApplied(t, db), len(migrations); got != want {
		t.Errorf("schema_migrations rows = %d, want %d", got, want)
	}

	// Spot-check a selection of application tables.
	for _, tbl := range []string{
		"accounts", "contacts", "bills", "invoices",
		"transactions", "transaction_documents", "payouts",
		"recurring_payments", "recurring_payment_occurrences",
		"bill_items", "invoice_items",
	} {
		if !tableExists(t, db, tbl) {
			t.Errorf("expected table %q to exist after MigrateDB", tbl)
		}
	}
}

// TestMigrateDB_Idempotent verifies that calling MigrateDB a second time does
// not apply migrations again or return an error.
func TestMigrateDB_Idempotent(t *testing.T) {
	db := openTestDB(t)

	if err := MigrateDB(db); err != nil {
		t.Fatalf("first MigrateDB: %v", err)
	}
	firstCount := countApplied(t, db)

	if err := MigrateDB(db); err != nil {
		t.Fatalf("second MigrateDB: %v", err)
	}
	secondCount := countApplied(t, db)

	if firstCount != secondCount {
		t.Errorf("schema_migrations count changed on second run: %d → %d", firstCount, secondCount)
	}
}

// TestRollbackDB_LastN verifies that RollbackDB(db, n) rolls back exactly the
// last n applied migrations in reverse order and removes their tracking rows.
func TestRollbackDB_LastN(t *testing.T) {
	db := openTestDB(t)

	if err := MigrateDB(db); err != nil {
		t.Fatalf("MigrateDB: %v", err)
	}
	total := countApplied(t, db)

	rollbackN := 3
	if err := RollbackDB(db, rollbackN); err != nil {
		t.Fatalf("RollbackDB(%d): %v", rollbackN, err)
	}

	if got, want := countApplied(t, db), total-rollbackN; got != want {
		t.Errorf("after rollback %d: schema_migrations rows = %d, want %d", rollbackN, got, want)
	}

	// The last rollbackN tables should no longer exist.
	for i := len(migrations) - 1; i >= len(migrations)-rollbackN; i-- {
		m := migrations[i]
		// Derive table name from the first word after "CREATE TABLE IF NOT EXISTS ".
		tbl := tableNameFromUp(m.Up)
		if tbl != "" && tableExists(t, db, tbl) {
			t.Errorf("table %q should not exist after rolling back migration %d", tbl, m.Version)
		}
	}
}

// TestRollbackDB_All verifies that RollbackDB(db, 0) rolls back every applied
// migration.
func TestRollbackDB_All(t *testing.T) {
	db := openTestDB(t)

	if err := MigrateDB(db); err != nil {
		t.Fatalf("MigrateDB: %v", err)
	}

	if err := RollbackDB(db, 0); err != nil {
		t.Fatalf("RollbackDB(0): %v", err)
	}

	if got := countApplied(t, db); got != 0 {
		t.Errorf("expected 0 applied migrations after full rollback, got %d", got)
	}

	// No application tables should exist after a full rollback.
	for _, m := range migrations {
		tbl := tableNameFromUp(m.Up)
		if tbl != "" && tableExists(t, db, tbl) {
			t.Errorf("table %q should not exist after full rollback", tbl)
		}
	}
}

// TestRollbackDB_ThenReapply verifies that after a rollback the migrations can
// be re-applied cleanly.
func TestRollbackDB_ThenReapply(t *testing.T) {
	db := openTestDB(t)

	if err := MigrateDB(db); err != nil {
		t.Fatalf("MigrateDB: %v", err)
	}

	if err := RollbackDB(db, 0); err != nil {
		t.Fatalf("RollbackDB(0): %v", err)
	}

	if err := MigrateDB(db); err != nil {
		t.Fatalf("MigrateDB after rollback: %v", err)
	}

	if got, want := countApplied(t, db), len(migrations); got != want {
		t.Errorf("re-applied migrations count = %d, want %d", got, want)
	}
}

// TestMigrateDB_SchemaVersionsAreCorrect verifies that every declared migration
// version is recorded with the expected version number after MigrateDB.
func TestMigrateDB_SchemaVersionsAreCorrect(t *testing.T) {
	db := openTestDB(t)

	if err := MigrateDB(db); err != nil {
		t.Fatalf("MigrateDB: %v", err)
	}

	rows, err := db.DB.Query(`SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	defer rows.Close()

	var got []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	if len(got) != len(migrations) {
		t.Fatalf("expected %d recorded versions, got %d", len(migrations), len(got))
	}
	for i, m := range migrations {
		if got[i] != m.Version {
			t.Errorf("recorded version[%d] = %d, want %d", i, got[i], m.Version)
		}
	}
}

// tableNameFromUp extracts the ASCII table name from a CREATE TABLE IF NOT EXISTS
// statement so tests can check whether it was dropped on rollback.
// Table names in migrations are expected to contain only ASCII letters, digits,
// and underscores.
func tableNameFromUp(up string) string {
	const prefix = "CREATE TABLE IF NOT EXISTS "
	idx := strings.Index(up, prefix)
	if idx < 0 {
		return ""
	}
	rest := up[idx+len(prefix):]
	end := strings.IndexAny(rest, " \t\n(")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

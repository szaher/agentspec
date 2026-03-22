package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresBackend implements Backend, HealthChecker, Locker, Closer, BudgetStore, and VersionStore using PostgreSQL.
type PostgresBackend struct {
	pool      *pgxpool.Pool
	tableName string
}

// NewPostgresBackend creates a new PostgreSQL state backend.
// If table is empty, defaults to "agentspec_state".
func NewPostgresBackend(dsn string, table string) (*PostgresBackend, error) {
	if table == "" {
		table = "agentspec_state"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	b := &PostgresBackend{
		pool:      pool,
		tableName: table,
	}

	// Auto-create tables on first use
	if err := b.initTables(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("initialize tables: %w", err)
	}

	return b, nil
}

// initTables creates the required tables if they don't exist.
func (b *PostgresBackend) initTables(ctx context.Context) error {
	queries := []string{
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				fqn TEXT PRIMARY KEY,
				hash TEXT NOT NULL,
				status TEXT NOT NULL,
				last_applied TIMESTAMPTZ NOT NULL,
				adapter TEXT NOT NULL,
				error_msg TEXT,
				orphaned_at TIMESTAMPTZ
			)
		`, b.tableName),
		`
			CREATE TABLE IF NOT EXISTS agentspec_budgets (
				agent TEXT PRIMARY KEY,
				data JSONB NOT NULL
			)
		`,
		`
			CREATE TABLE IF NOT EXISTS agentspec_versions (
				agent TEXT NOT NULL,
				version INT NOT NULL,
				data JSONB NOT NULL,
				PRIMARY KEY (agent, version)
			)
		`,
	}

	for _, query := range queries {
		if _, err := b.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	return nil
}

// Load reads all state entries from the database.
func (b *PostgresBackend) Load() ([]Entry, error) {
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT fqn, hash, status, last_applied, adapter, error_msg, orphaned_at
		FROM %s
		ORDER BY fqn
	`, b.tableName)

	rows, err := b.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var errorMsg *string
		var orphanedAt *time.Time

		if err := rows.Scan(&e.FQN, &e.Hash, &e.Status, &e.LastApplied, &e.Adapter, &errorMsg, &orphanedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}

		if errorMsg != nil {
			e.Error = *errorMsg
		}
		if orphanedAt != nil {
			e.OrphanedAt = *orphanedAt
		}

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	return entries, nil
}

// Save writes all state entries to the database using upsert.
// Entries not in the new list are deleted.
func (b *PostgresBackend) Save(entries []Entry) error {
	ctx := context.Background()
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Delete all existing entries
	if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s", b.tableName)); err != nil {
		return fmt.Errorf("delete old entries: %w", err)
	}

	// Insert new entries
	if len(entries) > 0 {
		query := fmt.Sprintf(`
			INSERT INTO %s (fqn, hash, status, last_applied, adapter, error_msg, orphaned_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, b.tableName)

		for _, e := range entries {
			var errorMsg *string
			if e.Error != "" {
				errorMsg = &e.Error
			}
			var orphanedAt *time.Time
			if !e.OrphanedAt.IsZero() {
				orphanedAt = &e.OrphanedAt
			}

			if _, err := tx.Exec(ctx, query, e.FQN, e.Hash, e.Status, e.LastApplied, e.Adapter, errorMsg, orphanedAt); err != nil {
				return fmt.Errorf("insert entry %s: %w", e.FQN, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Get retrieves a single entry by FQN.
func (b *PostgresBackend) Get(fqn string) (*Entry, error) {
	ctx := context.Background()
	query := fmt.Sprintf(`
		SELECT fqn, hash, status, last_applied, adapter, error_msg, orphaned_at
		FROM %s
		WHERE fqn = $1
	`, b.tableName)

	var e Entry
	var errorMsg *string
	var orphanedAt *time.Time

	err := b.pool.QueryRow(ctx, query, fqn).Scan(&e.FQN, &e.Hash, &e.Status, &e.LastApplied, &e.Adapter, &errorMsg, &orphanedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query entry: %w", err)
	}

	if errorMsg != nil {
		e.Error = *errorMsg
	}
	if orphanedAt != nil {
		e.OrphanedAt = *orphanedAt
	}

	return &e, nil
}

// List returns all entries, optionally filtered by status.
func (b *PostgresBackend) List(status *Status) ([]Entry, error) {
	ctx := context.Background()
	var query string
	var rows pgx.Rows
	var err error

	if status == nil {
		query = fmt.Sprintf(`
			SELECT fqn, hash, status, last_applied, adapter, error_msg, orphaned_at
			FROM %s
			ORDER BY fqn
		`, b.tableName)
		rows, err = b.pool.Query(ctx, query)
	} else {
		query = fmt.Sprintf(`
			SELECT fqn, hash, status, last_applied, adapter, error_msg, orphaned_at
			FROM %s
			WHERE status = $1
			ORDER BY fqn
		`, b.tableName)
		rows, err = b.pool.Query(ctx, query, *status)
	}

	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var errorMsg *string
		var orphanedAt *time.Time

		if err := rows.Scan(&e.FQN, &e.Hash, &e.Status, &e.LastApplied, &e.Adapter, &errorMsg, &orphanedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}

		if errorMsg != nil {
			e.Error = *errorMsg
		}
		if orphanedAt != nil {
			e.OrphanedAt = *orphanedAt
		}

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	return entries, nil
}

// Ping checks database connectivity.
func (b *PostgresBackend) Ping(ctx context.Context) error {
	if err := b.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping failed: %w", err)
	}
	return nil
}

// Lock acquires a PostgreSQL advisory lock.
func (b *PostgresBackend) Lock(ctx context.Context) error {
	query := `SELECT pg_advisory_lock(hashtext('agentspec_state'))`
	if _, err := b.pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	return nil
}

// Unlock releases the PostgreSQL advisory lock.
func (b *PostgresBackend) Unlock() error {
	ctx := context.Background()
	query := `SELECT pg_advisory_unlock(hashtext('agentspec_state'))`
	if _, err := b.pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("release advisory lock: %w", err)
	}
	return nil
}

// Close closes the connection pool.
func (b *PostgresBackend) Close() error {
	b.pool.Close()
	return nil
}

// LoadBudgets reads budget state from the database.
func (b *PostgresBackend) LoadBudgets() ([]BudgetState, error) {
	ctx := context.Background()
	query := `SELECT agent, data FROM agentspec_budgets ORDER BY agent`

	rows, err := b.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query budgets: %w", err)
	}
	defer rows.Close()

	var budgets []BudgetState
	for rows.Next() {
		var agent string
		var data []byte

		if err := rows.Scan(&agent, &data); err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}

		var budget BudgetState
		if err := json.Unmarshal(data, &budget); err != nil {
			return nil, fmt.Errorf("unmarshal budget for %s: %w", agent, err)
		}

		budgets = append(budgets, budget)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate budgets: %w", err)
	}

	return budgets, nil
}

// SaveBudgets writes budget state to the database.
func (b *PostgresBackend) SaveBudgets(budgets []BudgetState) error {
	ctx := context.Background()
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Clear existing budgets
	if _, err := tx.Exec(ctx, `DELETE FROM agentspec_budgets`); err != nil {
		return fmt.Errorf("delete old budgets: %w", err)
	}

	// Insert new budgets
	if len(budgets) > 0 {
		query := `INSERT INTO agentspec_budgets (agent, data) VALUES ($1, $2)`

		for _, budget := range budgets {
			data, err := json.Marshal(budget)
			if err != nil {
				return fmt.Errorf("marshal budget for %s: %w", budget.Agent, err)
			}

			if _, err := tx.Exec(ctx, query, budget.Agent, data); err != nil {
				return fmt.Errorf("insert budget for %s: %w", budget.Agent, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// SaveVersion records a new version entry for an agent.
func (b *PostgresBackend) SaveVersion(agent string, entry VersionEntry) error {
	ctx := context.Background()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal version entry: %w", err)
	}

	query := `
		INSERT INTO agentspec_versions (agent, version, data)
		VALUES ($1, $2, $3)
		ON CONFLICT (agent, version) DO UPDATE SET data = EXCLUDED.data
	`

	if _, err := b.pool.Exec(ctx, query, agent, entry.Version, data); err != nil {
		return fmt.Errorf("insert version for %s: %w", agent, err)
	}

	// Retain only the last 10 versions
	deleteQuery := `
		DELETE FROM agentspec_versions
		WHERE agent = $1
		AND version NOT IN (
			SELECT version FROM agentspec_versions
			WHERE agent = $1
			ORDER BY version DESC
			LIMIT 10
		)
	`

	if _, err := b.pool.Exec(ctx, deleteQuery, agent); err != nil {
		return fmt.Errorf("prune old versions for %s: %w", agent, err)
	}

	return nil
}

// GetVersions returns the version history for an agent.
func (b *PostgresBackend) GetVersions(agent string) ([]VersionEntry, error) {
	ctx := context.Background()
	query := `
		SELECT data FROM agentspec_versions
		WHERE agent = $1
		ORDER BY version ASC
	`

	rows, err := b.pool.Query(ctx, query, agent)
	if err != nil {
		return nil, fmt.Errorf("query versions: %w", err)
	}
	defer rows.Close()

	var versions []VersionEntry
	for rows.Next() {
		var data []byte

		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}

		var version VersionEntry
		if err := json.Unmarshal(data, &version); err != nil {
			return nil, fmt.Errorf("unmarshal version for %s: %w", agent, err)
		}

		versions = append(versions, version)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versions: %w", err)
	}

	return versions, nil
}

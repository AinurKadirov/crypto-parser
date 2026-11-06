package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps SQLite connection and migrations.
type DB struct {
	conn *sql.DB
}

// Open opens or creates the SQLite database and runs migrations.
func Open(dsn string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir results: %w", err)
	}

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS backtest_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    strategy TEXT NOT NULL,
    symbol TEXT NOT NULL,
    exchange TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    date_from TEXT NOT NULL,
    date_to TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    status TEXT NOT NULL DEFAULT 'pending'
);

CREATE TABLE IF NOT EXISTS backtest_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL REFERENCES backtest_runs(id),
    total_return_pct REAL,
    sharpe_ratio REAL,
    max_drawdown_pct REAL,
    win_rate_pct REAL,
    profit_factor REAL,
    total_trades INTEGER,
    final_balance REAL,
    params_json TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS param_sweeps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL REFERENCES backtest_runs(id),
    rank INTEGER NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    total_return_pct REAL,
    sharpe_ratio REAL,
    max_drawdown_pct REAL,
    win_rate_pct REAL,
    profit_factor REAL,
    total_trades INTEGER,
    final_balance REAL,
    params_json TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`
	_, err := db.conn.Exec(schema)
	return err
}

// RunSummary is a lightweight run record for the UI.
type RunSummary struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Strategy  string    `json:"strategy"`
	Symbol    string    `json:"symbol"`
	Timeframe string    `json:"timeframe"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ListRuns returns recent backtest runs.
func (db *DB) ListRuns(ctx context.Context, limit int) ([]RunSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, name, strategy, symbol, timeframe, status, created_at
		FROM backtest_runs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []RunSummary
	for rows.Next() {
		var r RunSummary
		var created string
		if err := rows.Scan(&r.ID, &r.Name, &r.Strategy, &r.Symbol, &r.Timeframe, &r.Status, &created); err != nil {
			return nil, err
		}
		r.CreatedAt, _ = parseSQLiteTime(created)
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// Ping checks database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

package sqlite

import (
	"context"
	"encoding/json"
	"math"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// RunInput describes a backtest run to persist.
type RunInput struct {
	Name      string
	Strategy  string
	Symbol    string
	Exchange  string
	Timeframe string
	DateFrom  string
	DateTo    string
}

// RunWithMetrics combines run metadata with its latest result.
type RunWithMetrics struct {
	RunSummary
	TotalReturnPct float64 `json:"total_return_pct"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	WinRatePct     float64 `json:"win_rate_pct"`
	ProfitFactor   float64 `json:"profit_factor"`
	TotalTrades    int     `json:"total_trades"`
	FinalBalance   float64 `json:"final_balance"`
	ParamsJSON     string  `json:"params_json,omitempty"`
}

// SaveRunResult stores a completed backtest run and its metrics.
func (db *DB) SaveRunResult(ctx context.Context, input RunInput, metrics domain.BacktestMetrics, params map[string]any) (int64, error) {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO backtest_runs (name, strategy, symbol, exchange, timeframe, date_from, date_to, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'completed')`,
		input.Name, input.Strategy, input.Symbol, input.Exchange, input.Timeframe, input.DateFrom, input.DateTo)
	if err != nil {
		return 0, err
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	paramsJSON, _ := json.Marshal(params)
	pf := metrics.ProfitFactor
	if math.IsInf(pf, 1) {
		pf = 999
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO backtest_results
		(run_id, total_return_pct, sharpe_ratio, max_drawdown_pct, win_rate_pct, profit_factor, total_trades, final_balance, params_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID,
		metrics.TotalReturnPct,
		metrics.SharpeRatio,
		metrics.MaxDrawdownPct,
		metrics.WinRatePct,
		pf,
		metrics.TotalTrades,
		metrics.FinalBalance,
		string(paramsJSON),
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return runID, nil
}

// ListRunsWithMetrics returns runs joined with their latest result.
func (db *DB) ListRunsWithMetrics(ctx context.Context, limit int) ([]RunWithMetrics, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT r.id, r.name, r.strategy, r.symbol, r.timeframe, r.status, r.created_at,
		       res.total_return_pct, res.sharpe_ratio, res.max_drawdown_pct,
		       res.win_rate_pct, res.profit_factor, res.total_trades, res.final_balance, res.params_json
		FROM backtest_runs r
		LEFT JOIN backtest_results res ON res.id = (
			SELECT id FROM backtest_results WHERE run_id = r.id ORDER BY id DESC LIMIT 1
		)
		ORDER BY r.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RunWithMetrics
	for rows.Next() {
		var item RunWithMetrics
		var created string
		var paramsJSON *string
		var ret, sharpe, dd, wr, pf, final *float64
		var trades *int

		if err := rows.Scan(
			&item.ID, &item.Name, &item.Strategy, &item.Symbol, &item.Timeframe, &item.Status, &created,
			&ret, &sharpe, &dd, &wr, &pf, &trades, &final, &paramsJSON,
		); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = parseSQLiteTime(created)
		if ret != nil {
			item.TotalReturnPct = *ret
		}
		if sharpe != nil {
			item.SharpeRatio = *sharpe
		}
		if dd != nil {
			item.MaxDrawdownPct = *dd
		}
		if wr != nil {
			item.WinRatePct = *wr
		}
		if pf != nil {
			item.ProfitFactor = *pf
		}
		if trades != nil {
			item.TotalTrades = *trades
		}
		if final != nil {
			item.FinalBalance = *final
		}
		if paramsJSON != nil {
			item.ParamsJSON = *paramsJSON
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

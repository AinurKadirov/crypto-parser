package sqlite

import (
	"context"
	"encoding/json"

	"github.com/AinurKadirov/crypto-parser/internal/optimize"
)

// SweepRow is one optimizer leaderboard row for UI/API.
type SweepRow struct {
	ID             int64   `json:"id"`
	RunID          int64   `json:"run_id"`
	Rank           int     `json:"rank"`
	MetricName     string  `json:"metric_name"`
	MetricValue    float64 `json:"metric_value"`
	Strategy       string  `json:"strategy"`
	Symbol         string  `json:"symbol"`
	Timeframe      string  `json:"timeframe"`
	TotalReturnPct float64 `json:"total_return_pct"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	TotalTrades    int     `json:"total_trades"`
	FinalBalance   float64 `json:"final_balance"`
	ParamsJSON     string  `json:"params_json"`
}

// SaveParamSweeps stores optimizer leaderboard rows.
func (db *DB) SaveParamSweeps(ctx context.Context, runID int64, metric string, sweeps []optimize.SweepResult) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO param_sweeps
		(run_id, rank, metric_name, metric_value, total_return_pct, sharpe_ratio, max_drawdown_pct, win_rate_pct, profit_factor, total_trades, final_balance, params_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, s := range sweeps {
		blob, _ := json.Marshal(s.Params)
		if _, err := stmt.ExecContext(
			ctx,
			runID,
			i+1,
			metric,
			s.MetricValue,
			s.Metrics.TotalReturnPct,
			s.Metrics.SharpeRatio,
			s.Metrics.MaxDrawdownPct,
			s.Metrics.WinRatePct,
			s.Metrics.ProfitFactor,
			s.Metrics.TotalTrades,
			s.Metrics.FinalBalance,
			string(blob),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListSweeps returns recent optimization leaderboard rows.
func (db *DB) ListSweeps(ctx context.Context, limit int) ([]SweepRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT s.id, s.run_id, s.rank, s.metric_name, s.metric_value,
		       r.strategy, r.symbol, r.timeframe,
		       s.total_return_pct, s.sharpe_ratio, s.max_drawdown_pct, s.total_trades, s.final_balance, s.params_json
		FROM param_sweeps s
		JOIN backtest_runs r ON r.id = s.run_id
		ORDER BY s.id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SweepRow
	for rows.Next() {
		var item SweepRow
		if err := rows.Scan(
			&item.ID, &item.RunID, &item.Rank, &item.MetricName, &item.MetricValue,
			&item.Strategy, &item.Symbol, &item.Timeframe,
			&item.TotalReturnPct, &item.SharpeRatio, &item.MaxDrawdownPct, &item.TotalTrades, &item.FinalBalance, &item.ParamsJSON,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

package backtest

import (
	"math"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// ComputeMetrics calculates aggregate statistics from a backtest run.
func ComputeMetrics(initial, final float64, trades []domain.Trade, equity []domain.EquityPoint) domain.BacktestMetrics {
	totalReturn := 0.0
	if initial > 0 {
		totalReturn = (final - initial) / initial * 100
	}

	wins, losses := 0, 0
	grossProfit, grossLoss := 0.0, 0.0
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
			grossProfit += t.PnL
		} else if t.PnL < 0 {
			losses++
			grossLoss += math.Abs(t.PnL)
		}
	}

	winRate := 0.0
	if len(trades) > 0 {
		winRate = float64(wins) / float64(len(trades)) * 100
	}

	pf := 0.0
	if grossLoss > 0 {
		pf = grossProfit / grossLoss
	} else if grossProfit > 0 {
		pf = math.Inf(1)
	}

	return domain.BacktestMetrics{
		TotalReturnPct: totalReturn,
		SharpeRatio:    sharpeFromEquity(equity),
		MaxDrawdownPct: maxDrawdown(equity),
		WinRatePct:     winRate,
		ProfitFactor:   pf,
		TotalTrades:    len(trades),
		FinalBalance:   final,
	}
}

func maxDrawdown(equity []domain.EquityPoint) float64 {
	if len(equity) == 0 {
		return 0
	}
	peak := equity[0].Equity
	maxDD := 0.0
	for _, p := range equity {
		if p.Equity > peak {
			peak = p.Equity
		}
		if peak > 0 {
			dd := (peak - p.Equity) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD
}

func sharpeFromEquity(equity []domain.EquityPoint) float64 {
	if len(equity) < 2 {
		return 0
	}
	returns := make([]float64, 0, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		prev := equity[i-1].Equity
		if prev <= 0 {
			continue
		}
		returns = append(returns, (equity[i].Equity-prev)/prev)
	}
	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(returns) - 1)
	std := math.Sqrt(variance)
	if std == 0 {
		return 0
	}
	// Scale for hourly bars: ~8760 bars/year; use sqrt(252) as daily proxy for crypto hourly
	return mean / std * math.Sqrt(252)
}

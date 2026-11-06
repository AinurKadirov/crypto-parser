package backtest

import (
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
	"github.com/AinurKadirov/crypto-parser/internal/strategy"
)

// Engine simulates trades over historical bars.
type Engine struct {
	risk RiskConfig
}

// NewEngine creates a backtest engine with risk settings.
func NewEngine(risk RiskConfig) *Engine {
	if risk.InitialBalance <= 0 {
		risk.InitialBalance = 10000
	}
	if risk.MaxPositionPct <= 0 {
		risk.MaxPositionPct = 100
	}
	return &Engine{risk: risk}
}

// Run executes a backtest with the given strategy.
func (e *Engine) Run(bars []domain.Bar, strat strategy.Strategy) domain.BacktestResult {
	balance := e.risk.InitialBalance
	var position *domain.Position
	var trades []domain.Trade
	equity := make([]domain.EquityPoint, 0, len(bars))

	for i, bar := range bars {
		if position != nil {
			if exitPrice, reason, ok := e.checkExit(position, bar); ok {
				trade, proceeds := e.closeLong(position, exitPrice, bar.OpenTime, reason)
				balance += proceeds
				trades = append(trades, trade)
				position = nil
			}
		}

		ctx := &strategy.Context{
			Balance:  balance,
			Position: position,
			BarCount: len(bars),
		}
		sig := strat.OnBar(bar, i, ctx)

		if position != nil && (sig.Action == domain.ActionSell || sig.Action == domain.ActionClose) {
			exitPrice := bar.Close * (1 - e.risk.SlippagePct/100)
			trade, proceeds := e.closeLong(position, exitPrice, bar.OpenTime, sig.Reason)
			balance += proceeds
			trades = append(trades, trade)
			position = nil
		}

		if position == nil && sig.Action == domain.ActionBuy {
			pos, cost := e.openLong(balance, bar, sig)
			if pos != nil {
				position = pos
				balance -= cost
			}
		}

		eq := balance
		if position != nil {
			eq += position.Quantity * bar.Close
		}
		equity = append(equity, domain.EquityPoint{
			Time: bar.OpenTime, Balance: balance, Equity: eq,
		})
	}

	if position != nil && len(bars) > 0 {
		last := bars[len(bars)-1]
		trade, proceeds := e.closeLong(position, last.Close, last.CloseTime, "end_of_data")
		balance += proceeds
		trades = append(trades, trade)
	}

	metrics := ComputeMetrics(e.risk.InitialBalance, balance, trades, equity)
	return domain.BacktestResult{
		Metrics:        metrics,
		Trades:         trades,
		EquityCurve:    equity,
		InitialBalance: e.risk.InitialBalance,
		FinalBalance:   balance,
	}
}

func (e *Engine) openLong(balance float64, bar domain.Bar, sig domain.Signal) (*domain.Position, float64) {
	sizePct := sig.SizePct
	if sizePct <= 0 {
		sizePct = e.risk.MaxPositionPct
	}
	if sizePct > e.risk.MaxPositionPct {
		sizePct = e.risk.MaxPositionPct
	}

	alloc := balance * sizePct / 100
	if alloc <= 0 {
		return nil, 0
	}

	entry := bar.Close * (1 + e.risk.SlippagePct/100)
	qty := alloc / entry
	notional := entry * qty
	commission := notional * e.risk.CommissionPct / 100
	totalCost := notional + commission

	pos := &domain.Position{
		Side:       domain.SideLong,
		EntryPrice: entry,
		Quantity:   qty,
		EntryTime:  bar.OpenTime,
	}

	slPct := sig.StopLoss
	if slPct <= 0 {
		slPct = e.risk.StopLossPct
	}
	tpPct := sig.TakeProfit
	if tpPct <= 0 {
		tpPct = e.risk.TakeProfitPct
	}
	if slPct > 0 {
		pos.StopLoss = entry * (1 - slPct/100)
	}
	if tpPct > 0 {
		pos.TakeProfit = entry * (1 + tpPct/100)
	}
	return pos, totalCost
}

func (e *Engine) checkExit(pos *domain.Position, bar domain.Bar) (price float64, reason string, ok bool) {
	if pos.Side != domain.SideLong {
		return 0, "", false
	}
	if pos.StopLoss > 0 && bar.Low <= pos.StopLoss {
		return pos.StopLoss * (1 - e.risk.SlippagePct/100), "stop_loss", true
	}
	if pos.TakeProfit > 0 && bar.High >= pos.TakeProfit {
		return pos.TakeProfit * (1 - e.risk.SlippagePct/100), "take_profit", true
	}
	return 0, "", false
}

func (e *Engine) closeLong(pos *domain.Position, exitPrice float64, exitTime time.Time, reason string) (domain.Trade, float64) {
	proceeds := exitPrice * pos.Quantity
	commission := proceeds * e.risk.CommissionPct / 100
	net := proceeds - commission

	cost := pos.EntryPrice * pos.Quantity
	entryCommission := cost * e.risk.CommissionPct / 100
	pnl := net - (cost + entryCommission)
	pnlPct := 0.0
	if cost+entryCommission > 0 {
		pnlPct = pnl / (cost + entryCommission) * 100
	}

	trade := domain.Trade{
		Side:       pos.Side,
		EntryPrice: pos.EntryPrice,
		ExitPrice:  exitPrice,
		Quantity:   pos.Quantity,
		EntryTime:  pos.EntryTime,
		ExitTime:   exitTime,
		PnL:        pnl,
		PnLPct:     pnlPct,
		Reason:     reason,
	}
	return trade, net
}

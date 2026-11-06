package domain

import "time"

// Side is position direction.
type Side string

const (
	SideLong  Side = "long"
	SideShort Side = "short"
)

// Position is an open market position.
type Position struct {
	Side       Side
	EntryPrice float64
	Quantity   float64
	EntryTime  time.Time
	StopLoss   float64
	TakeProfit float64
}

// Trade is a completed round-trip.
type Trade struct {
	Side       Side
	EntryPrice float64
	ExitPrice  float64
	Quantity   float64
	EntryTime  time.Time
	ExitTime   time.Time
	PnL        float64
	PnLPct     float64
	Reason     string
}

// EquityPoint is balance snapshot at a bar.
type EquityPoint struct {
	Time    time.Time
	Balance float64
	Equity  float64
}

// BacktestResult is the full output of a backtest run.
type BacktestResult struct {
	Metrics      BacktestMetrics
	Trades       []Trade
	EquityCurve  []EquityPoint
	InitialBalance float64
	FinalBalance   float64
}

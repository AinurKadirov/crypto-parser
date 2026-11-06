package domain

import "time"

// Bar represents a single OHLCV candlestick.
type Bar struct {
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

// SymbolKey uniquely identifies a market series.
type SymbolKey struct {
	Exchange  string
	Symbol    string
	Timeframe string
}

// DataInventory describes stored candle data metadata.
type DataInventory struct {
	Key       SymbolKey
	BarCount  int
	FirstBar  time.Time
	LastBar   time.Time
	FilePath  string
	FileSizeB int64
}

// Action represents a trading signal action.
type Action string

const (
	ActionHold  Action = "hold"
	ActionBuy   Action = "buy"
	ActionSell  Action = "sell"
	ActionClose Action = "close"
)

// Signal is emitted by a strategy on each bar.
type Signal struct {
	Action     Action
	SizePct    float64
	StopLoss   float64
	TakeProfit float64
	Reason     string
}

// BacktestMetrics holds aggregated backtest statistics.
type BacktestMetrics struct {
	TotalReturnPct float64
	SharpeRatio    float64
	MaxDrawdownPct float64
	WinRatePct     float64
	ProfitFactor   float64
	TotalTrades    int
	FinalBalance   float64
}

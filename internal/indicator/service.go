package indicator

import (
	"context"
	"math"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// Snapshot is a point-in-time view of all indicators for the last bar.
type Snapshot struct {
	Timestamp time.Time `json:"timestamp"`
	Close     float64   `json:"close"`
	EMA12     float64   `json:"ema_12"`
	EMA26     float64   `json:"ema_26"`
	SMA20     float64   `json:"sma_20"`
	RSI14     float64   `json:"rsi_14"`
	MACD      float64   `json:"macd"`
	MACDSignal float64  `json:"macd_signal"`
	VolumeSMA float64   `json:"volume_sma_20"`
}

// Preview holds recent indicator snapshots for UI.
type Preview struct {
	Symbol    string     `json:"symbol"`
	Timeframe string     `json:"timeframe"`
	BarCount  int        `json:"bar_count"`
	Latest    Snapshot   `json:"latest"`
	Recent    []Snapshot `json:"recent"`
}

// Service computes indicators over stored candles.
type Service struct{}

// NewService creates an indicator service.
func NewService() *Service {
	return &Service{}
}

// PreviewFromBars computes standard indicators and returns last N snapshots.
func (s *Service) PreviewFromBars(_ context.Context, symbol, timeframe string, bars []domain.Bar, tail int) Preview {
	if tail <= 0 {
		tail = 20
	}
	input := BarsFromDomain(bars)
	closes := ClosesFromBars(input)

	ema12 := EMA{Period: 12}.Compute(closes)
	ema26 := EMA{Period: 26}.Compute(closes)
	sma20 := SMA{Period: 20}.Compute(closes)
	rsi14 := RSI{Period: 14}.Compute(closes)
	macd := MACD{}.ComputeFull(closes)
	volSMA := VolumeSMA{Period: 20}.ComputeBars(input)

	n := len(bars)
	start := n - tail
	if start < 0 {
		start = 0
	}

	recent := make([]Snapshot, 0, n-start)
	for i := start; i < n; i++ {
		recent = append(recent, Snapshot{
			Timestamp:  bars[i].OpenTime,
			Close:      bars[i].Close,
			EMA12:      safeVal(ema12.Values[i]),
			EMA26:      safeVal(ema26.Values[i]),
			SMA20:      safeVal(sma20.Values[i]),
			RSI14:      safeVal(rsi14.Values[i]),
			MACD:       safeVal(macd.MACD.Values[i]),
			MACDSignal: safeVal(macd.Signal.Values[i]),
			VolumeSMA:  safeVal(volSMA.Values[i]),
		})
	}

	var latest Snapshot
	if len(recent) > 0 {
		latest = recent[len(recent)-1]
	}

	return Preview{
		Symbol:    symbol,
		Timeframe: timeframe,
		BarCount:  n,
		Latest:    latest,
		Recent:    recent,
	}
}

func safeVal(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return round4(v)
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

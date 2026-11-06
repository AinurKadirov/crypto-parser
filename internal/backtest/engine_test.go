package backtest

import (
	"math"
	"testing"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
	"github.com/AinurKadirov/crypto-parser/internal/strategy"
)

func barsWithPrices(prices []float64) []domain.Bar {
	out := make([]domain.Bar, len(prices))
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, p := range prices {
		out[i] = domain.Bar{
			OpenTime:  t0.Add(time.Duration(i) * time.Hour),
			Open:      p,
			High:      p * 1.001,
			Low:       p * 0.999,
			Close:     p,
			CloseTime: t0.Add(time.Duration(i+1) * time.Hour),
		}
	}
	return out
}

func TestBuyAndHold_returnMatchesPriceChange(t *testing.T) {
	bars := barsWithPrices([]float64{100, 100, 110})
	risk := RiskConfig{
		InitialBalance: 10000,
		CommissionPct:  0,
		SlippagePct:    0,
		MaxPositionPct: 100,
		StopLossPct:    0,
		TakeProfitPct:  0,
	}
	result := NewEngine(risk).Run(bars, strategy.BuyAndHold{})
	// 100 -> 110 = +10% minus nothing
	if math.Abs(result.Metrics.TotalReturnPct-10) > 0.5 {
		t.Fatalf("return=%v want ~10", result.Metrics.TotalReturnPct)
	}
}

func TestStopLoss_triggersOnLow(t *testing.T) {
	bars := []domain.Bar{
		{OpenTime: time.Now(), Close: 100, High: 101, Low: 99},
		{OpenTime: time.Now().Add(time.Hour), Close: 97, High: 98, Low: 97},
	}
	bars[1].Low = 97 // below SL 98 for entry ~100 with 2% SL

	risk := RiskConfig{
		InitialBalance: 10000,
		CommissionPct:  0,
		SlippagePct:    0,
		StopLossPct:    2,
		TakeProfitPct:  0,
		MaxPositionPct: 100,
	}
	result := NewEngine(risk).Run(bars, strategy.BuyAndHold{})
	if len(result.Trades) == 0 {
		t.Fatal("expected stop loss trade")
	}
	if result.Trades[0].Reason != "stop_loss" {
		t.Fatalf("reason=%s want stop_loss", result.Trades[0].Reason)
	}
}

func TestMetrics_profitFactor(t *testing.T) {
	trades := []domain.Trade{
		{PnL: 100}, {PnL: -50},
	}
	m := ComputeMetrics(10000, 10050, trades, nil)
	if math.Abs(m.ProfitFactor-2) > 1e-9 {
		t.Fatalf("pf=%v want 2", m.ProfitFactor)
	}
	if m.WinRatePct != 50 {
		t.Fatalf("winrate=%v want 50", m.WinRatePct)
	}
}

func TestMaxDrawdown(t *testing.T) {
	eq := []domain.EquityPoint{
		{Equity: 100}, {Equity: 120}, {Equity: 90},
	}
	dd := maxDrawdown(eq)
	want := (120.0 - 90.0) / 120.0 * 100
	if math.Abs(dd-want) > 1e-9 {
		t.Fatalf("dd=%v want %v", dd, want)
	}
}

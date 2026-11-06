package backtest

// RiskConfig controls position sizing and execution costs.
type RiskConfig struct {
	InitialBalance float64
	CommissionPct  float64
	SlippagePct    float64
	StopLossPct    float64
	TakeProfitPct  float64
	MaxPositionPct float64
}

// DefaultRisk returns sensible defaults matching configs/strategies/ema_crossover.yaml.
func DefaultRisk() RiskConfig {
	return RiskConfig{
		InitialBalance: 10000,
		CommissionPct:  0.1,
		SlippagePct:    0.05,
		StopLossPct:    2.0,
		TakeProfitPct:  4.0,
		MaxPositionPct: 100,
	}
}

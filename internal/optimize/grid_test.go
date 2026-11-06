package optimize

import (
	"context"
	"testing"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/backtest"
	"github.com/AinurKadirov/crypto-parser/internal/config"
	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

func TestBuildGridCandidates(t *testing.T) {
	cfg := &config.Strategy{
		Parameters: map[string]config.ParamDef{
			"ema_fast": {Type: "int", Range: []float64{5, 7}, Step: 1},
			"ema_slow": {Type: "int", Range: []float64{20, 22}, Step: 1},
		},
	}
	base := map[string]any{"volume_multiplier": 1.5}
	cands := BuildGridCandidates(cfg, base, 0)
	if len(cands) != 9 {
		t.Fatalf("want 9 candidates, got %d", len(cands))
	}
}

func TestRunnerEvaluate(t *testing.T) {
	bars := []domain.Bar{
		{OpenTime: time.Now(), Close: 100, High: 101, Low: 99, Volume: 1000},
		{OpenTime: time.Now().Add(time.Hour), Close: 101, High: 102, Low: 100, Volume: 1200},
		{OpenTime: time.Now().Add(2 * time.Hour), Close: 102, High: 103, Low: 101, Volume: 1300},
		{OpenTime: time.Now().Add(3 * time.Hour), Close: 103, High: 104, Low: 102, Volume: 1400},
	}
	cfg := &config.Strategy{
		Parameters: map[string]config.ParamDef{
			"ema_fast":          {Type: "int", Range: []float64{2, 3}, Step: 1},
			"ema_slow":          {Type: "int", Range: []float64{4, 5}, Step: 1},
			"volume_multiplier": {Type: "float", Range: []float64{1.0, 1.2}, Step: 0.2},
		},
	}
	cands := BuildGridCandidates(cfg, map[string]any{}, 5)

	r := &Runner{
		Bars:     bars,
		Risk:     backtest.DefaultRisk(),
		Strategy: "ema_crossover",
		StratCfg: cfg,
		Metric:   "sharpe_ratio",
		Workers:  2,
	}
	out, err := r.Evaluate(context.Background(), cands)
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if len(out) != len(cands) {
		t.Fatalf("want %d results got %d", len(cands), len(out))
	}
}

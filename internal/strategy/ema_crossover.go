package strategy

import (
	"fmt"
	"math"

	"github.com/AinurKadirov/crypto-parser/internal/config"
	"github.com/AinurKadirov/crypto-parser/internal/domain"
	"github.com/AinurKadirov/crypto-parser/internal/indicator"
)

const defaultVolumeSMAPeriod = 20

// EMACrossover trades fast/slow EMA crosses with volume confirmation.
type EMACrossover struct {
	emaFast   []float64
	emaSlow   []float64
	volSMA    []float64
	volumes   []float64
	volMult   float64
	sizePct   float64
}

// EMACrossoverParams holds strategy tuning values.
type EMACrossoverParams struct {
	EMAFast           int
	EMASlow           int
	VolumeMultiplier  float64
	VolumeSMAPeriod   int
	MaxPositionPct    float64
}

// NewEMACrossover builds a prepared strategy from historical bars.
func NewEMACrossover(bars []domain.Bar, p EMACrossoverParams) (*EMACrossover, error) {
	if p.EMAFast <= 0 {
		p.EMAFast = 12
	}
	if p.EMASlow <= 0 {
		p.EMASlow = 26
	}
	if p.EMAFast >= p.EMASlow {
		return nil, fmt.Errorf("ema_fast (%d) must be < ema_slow (%d)", p.EMAFast, p.EMASlow)
	}
	if p.VolumeMultiplier <= 0 {
		p.VolumeMultiplier = 1.5
	}
	if p.VolumeSMAPeriod <= 0 {
		p.VolumeSMAPeriod = defaultVolumeSMAPeriod
	}
	if p.MaxPositionPct <= 0 {
		p.MaxPositionPct = 10
	}

	input := indicator.BarsFromDomain(bars)
	closes := indicator.ClosesFromBars(input)

	s := &EMACrossover{
		emaFast: indicator.EMA{Period: p.EMAFast}.Compute(closes).Values,
		emaSlow: indicator.EMA{Period: p.EMASlow}.Compute(closes).Values,
		volSMA:  indicator.VolumeSMA{Period: p.VolumeSMAPeriod}.ComputeBars(input).Values,
		volumes: indicator.VolumesFromBars(input),
		volMult: p.VolumeMultiplier,
		sizePct: p.MaxPositionPct,
	}
	return s, nil
}

func (s *EMACrossover) Name() string { return "ema_crossover" }

func (s *EMACrossover) OnBar(_ domain.Bar, index int, ctx *Context) domain.Signal {
	if index < 1 || index >= len(s.emaFast) {
		return domain.Signal{Action: domain.ActionHold}
	}

	if ctx.HasPosition() {
		if crossedBelow(s.emaFast, s.emaSlow, index) {
			return domain.Signal{Action: domain.ActionClose, Reason: "ema_bearish_cross"}
		}
		return domain.Signal{Action: domain.ActionHold}
	}

	if crossedAbove(s.emaFast, s.emaSlow, index) && s.volumeOK(index) {
		return domain.Signal{
			Action:  domain.ActionBuy,
			SizePct: s.sizePct,
			Reason:  "ema_bullish_cross",
		}
	}
	return domain.Signal{Action: domain.ActionHold}
}

func (s *EMACrossover) volumeOK(i int) bool {
	if i >= len(s.volSMA) || i >= len(s.volumes) {
		return false
	}
	vsma := s.volSMA[i]
	if !valid(vsma) || vsma <= 0 {
		return false
	}
	return s.volumes[i] > vsma*s.volMult
}

func crossedAbove(fast, slow []float64, i int) bool {
	if !valid(fast[i]) || !valid(slow[i]) || !valid(fast[i-1]) || !valid(slow[i-1]) {
		return false
	}
	return fast[i-1] <= slow[i-1] && fast[i] > slow[i]
}

func crossedBelow(fast, slow []float64, i int) bool {
	if !valid(fast[i]) || !valid(slow[i]) || !valid(fast[i-1]) || !valid(slow[i-1]) {
		return false
	}
	return fast[i-1] >= slow[i-1] && fast[i] < slow[i]
}

func valid(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// ParamsFromConfig merges run overrides with strategy YAML defaults.
func ParamsFromConfig(stratCfg *config.Strategy, runParams map[string]any) EMACrossoverParams {
	p := EMACrossoverParams{
		EMAFast:          intParam(stratCfg, runParams, "ema_fast", 12),
		EMASlow:          intParam(stratCfg, runParams, "ema_slow", 26),
		VolumeMultiplier: floatParam(stratCfg, runParams, "volume_multiplier", 1.5),
	}
	if stratCfg != nil && stratCfg.Risk != nil {
		p.MaxPositionPct = stratCfg.Risk["max_position_pct"]
	}
	if p.MaxPositionPct <= 0 {
		p.MaxPositionPct = 10
	}
	return p
}

func intParam(cfg *config.Strategy, overrides map[string]any, key string, def int) int {
	if v, ok := overrides[key]; ok {
		if n := toInt(v); n > 0 {
			return n
		}
	}
	if cfg != nil {
		if p, ok := cfg.Parameters[key]; ok {
			if n := toInt(p.Default); n > 0 {
				return n
			}
		}
	}
	return def
}

func floatParam(cfg *config.Strategy, overrides map[string]any, key string, def float64) float64 {
	if v, ok := overrides[key]; ok {
		if f := toFloat(v); f > 0 {
			return f
		}
	}
	if cfg != nil {
		if p, ok := cfg.Parameters[key]; ok {
			if f := toFloat(p.Default); f > 0 {
				return f
			}
		}
	}
	return def
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

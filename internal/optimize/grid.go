package optimize

import (
	"context"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/backtest"
	"github.com/AinurKadirov/crypto-parser/internal/config"
	"github.com/AinurKadirov/crypto-parser/internal/domain"
	"github.com/AinurKadirov/crypto-parser/internal/strategy"
)

// Candidate is one parameter combination.
type Candidate map[string]any

// SweepResult is one evaluated optimization result.
type SweepResult struct {
	Params      map[string]any
	Metrics     domain.BacktestMetrics
	MetricName  string
	MetricValue float64
}

// Runner executes optimization jobs.
type Runner struct {
	Bars     []domain.Bar
	Risk     backtest.RiskConfig
	Strategy string
	StratCfg *config.Strategy
	Metric   string
	Workers  int
}

// BuildGridCandidates builds cartesian product from strategy parameter ranges.
func BuildGridCandidates(stratCfg *config.Strategy, base map[string]any, maxIterations int) []Candidate {
	candidates := []Candidate{{}}
	for k, v := range base {
		candidates[0][k] = v
	}
	if stratCfg == nil {
		return candidates
	}

	keys := make([]string, 0, len(stratCfg.Parameters))
	for key, def := range stratCfg.Parameters {
		if len(def.Range) == 2 && def.Step > 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	for _, key := range keys {
		def := stratCfg.Parameters[key]
		values := expandValues(def)
		if len(values) == 0 {
			continue
		}
		next := make([]Candidate, 0, len(candidates)*len(values))
		for _, c := range candidates {
			for _, value := range values {
				nc := cloneCandidate(c)
				nc[key] = castByType(value, def.Type)
				next = append(next, nc)
				if maxIterations > 0 && len(next) >= maxIterations {
					return next
				}
			}
		}
		candidates = next
	}
	if maxIterations > 0 && len(candidates) > maxIterations {
		return candidates[:maxIterations]
	}
	return candidates
}

// BuildRandomCandidates samples combinations from the same ranges.
func BuildRandomCandidates(stratCfg *config.Strategy, base map[string]any, maxIterations int) []Candidate {
	grid := BuildGridCandidates(stratCfg, base, 0)
	if maxIterations <= 0 || len(grid) <= maxIterations {
		return grid
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(grid), func(i, j int) { grid[i], grid[j] = grid[j], grid[i] })
	return grid[:maxIterations]
}

// Evaluate runs candidates using worker pool and returns sorted results.
func (r *Runner) Evaluate(ctx context.Context, candidates []Candidate) ([]SweepResult, error) {
	if r.Workers <= 0 {
		r.Workers = runtime.NumCPU()
	}
	if r.Workers < 1 {
		r.Workers = 1
	}

	type job struct {
		idx int
		c   Candidate
	}
	type out struct {
		idx int
		v   SweepResult
		err error
	}

	jobs := make(chan job)
	results := make(chan out, len(candidates))
	var wg sync.WaitGroup

	for i := 0; i < r.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg := strategy.NewRegistry()
			engine := backtest.NewEngine(r.Risk)
			for j := range jobs {
				if err := ctx.Err(); err != nil {
					results <- out{idx: j.idx, err: err}
					continue
				}
				strat, err := reg.Create(r.Strategy, r.Bars, r.StratCfg, j.c)
				if err != nil {
					results <- out{idx: j.idx, err: err}
					continue
				}
				bt := engine.Run(r.Bars, strat)
				score := metricScore(r.Metric, bt.Metrics)
				results <- out{
					idx: j.idx,
					v: SweepResult{
						Params:      cloneCandidate(j.c),
						Metrics:     bt.Metrics,
						MetricName:  r.Metric,
						MetricValue: score,
					},
				}
			}
		}()
	}

	go func() {
		for i, c := range candidates {
			jobs <- job{idx: i, c: c}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	outAll := make([]SweepResult, 0, len(candidates))
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		outAll = append(outAll, res.v)
	}

	sort.Slice(outAll, func(i, j int) bool {
		return outAll[i].MetricValue > outAll[j].MetricValue
	})
	return outAll, nil
}

func metricScore(metric string, m domain.BacktestMetrics) float64 {
	switch metric {
	case "total_return", "total_return_pct":
		return m.TotalReturnPct
	case "profit_factor":
		return m.ProfitFactor
	case "max_drawdown", "max_drawdown_pct":
		return -m.MaxDrawdownPct
	case "sharpe", "sharpe_ratio":
		fallthrough
	default:
		return m.SharpeRatio
	}
}

func expandValues(p config.ParamDef) []float64 {
	if len(p.Range) != 2 || p.Step <= 0 {
		return nil
	}
	minV := p.Range[0]
	maxV := p.Range[1]
	if maxV < minV {
		minV, maxV = maxV, minV
	}
	out := make([]float64, 0, 32)
	for v := minV; v <= maxV+1e-9; v += p.Step {
		out = append(out, v)
	}
	return out
}

func castByType(v float64, typ string) any {
	switch typ {
	case "int":
		return int(v)
	default:
		return v
	}
}

func cloneCandidate(c Candidate) map[string]any {
	out := make(map[string]any, len(c))
	for k, v := range c {
		out[k] = v
	}
	return out
}

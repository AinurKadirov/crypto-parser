package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/backtest"
	"github.com/AinurKadirov/crypto-parser/internal/config"
	"github.com/AinurKadirov/crypto-parser/internal/domain"
	"github.com/AinurKadirov/crypto-parser/internal/optimize"
	"github.com/AinurKadirov/crypto-parser/internal/storage/parquet"
	"github.com/AinurKadirov/crypto-parser/internal/storage/sqlite"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	runPath := flag.String("run", "configs/runs/btc_1h_ema.yaml", "Run config path")
	flag.Parse()

	loader := config.NewLoader("./configs")
	appCfg, err := loader.LoadApp()
	if err != nil {
		logger.Error("load app", "err", err)
		os.Exit(1)
	}
	loader = config.NewLoader(config.ResolveConfigDir(appCfg))

	run, err := loader.LoadRun(*runPath)
	if err != nil {
		logger.Error("load run", "err", err)
		os.Exit(1)
	}
	stratCfg, err := loader.LoadStrategy(run.Run.Strategy)
	if err != nil {
		logger.Error("load strategy", "err", err)
		os.Exit(1)
	}

	key := domain.SymbolKey{
		Exchange:  run.Run.Exchange,
		Symbol:    run.Run.Symbol,
		Timeframe: run.Run.Timeframe,
	}
	candleRepo := parquet.NewCandleRepo(config.ResolveDataDir(appCfg))
	bars, err := candleRepo.Load(context.Background(), key)
	if err != nil {
		logger.Error("load bars", "err", err)
		os.Exit(1)
	}
	if len(bars) == 0 {
		logger.Error("no data", "hint", "run fetch first")
		os.Exit(1)
	}

	from, _ := time.Parse("2006-01-02", run.Run.DateFrom)
	to, _ := time.Parse("2006-01-02", run.Run.DateTo)
	to = to.Add(24*time.Hour - time.Millisecond)
	bars = filterBars(bars, from, to)

	risk := backtest.RiskConfig{
		InitialBalance: getFloat(stratCfg.Backtest, "initial_balance", 10000),
		CommissionPct:  stratCfg.Risk["commission_pct"],
		SlippagePct:    stratCfg.Risk["slippage_pct"],
		StopLossPct:    stratCfg.Risk["stop_loss_pct"],
		TakeProfitPct:  stratCfg.Risk["take_profit_pct"],
		MaxPositionPct: stratCfg.Risk["max_position_pct"],
	}

	var candidates []optimize.Candidate
	if strings.EqualFold(run.Optimize.Method, "random") {
		candidates = optimize.BuildRandomCandidates(stratCfg, run.Params, run.Optimize.MaxIterations)
	} else {
		candidates = optimize.BuildGridCandidates(stratCfg, run.Params, run.Optimize.MaxIterations)
	}
	if len(candidates) == 0 {
		logger.Error("no candidates generated")
		os.Exit(1)
	}

	logger.Info("optimizer started",
		"strategy", run.Run.Strategy,
		"symbol", run.Run.Symbol,
		"timeframe", run.Run.Timeframe,
		"candidates", len(candidates),
		"metric", run.Optimize.Metric,
		"method", run.Optimize.Method,
	)

	worker := &optimize.Runner{
		Bars:     bars,
		Risk:     risk,
		Strategy: run.Run.Strategy,
		StratCfg: stratCfg,
		Metric:   run.Optimize.Metric,
	}
	results, err := worker.Evaluate(context.Background(), candidates)
	if err != nil {
		logger.Error("evaluate failed", "err", err)
		os.Exit(1)
	}

	topN := run.Optimize.TopN
	if topN <= 0 {
		topN = 10
	}
	if topN > len(results) {
		topN = len(results)
	}
	top := results[:topN]
	best := top[0]

	dsn := appCfg.Database.DSN
	if env := os.Getenv("CRYPTO_PARSER_DB_DSN"); env != "" {
		dsn = strings.TrimPrefix(env, "sqlite:")
	}
	db, err := sqlite.Open(dsn)
	if err != nil {
		logger.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	runName := run.Run.Name + "_optimize"
	runID, err := db.SaveRunResult(context.Background(), sqlite.RunInput{
		Name:      runName,
		Strategy:  run.Run.Strategy,
		Symbol:    run.Run.Symbol,
		Exchange:  run.Run.Exchange,
		Timeframe: run.Run.Timeframe,
		DateFrom:  run.Run.DateFrom,
		DateTo:    run.Run.DateTo,
	}, best.Metrics, best.Params)
	if err != nil {
		logger.Error("save optimization run", "err", err)
		os.Exit(1)
	}
	if err := db.SaveParamSweeps(context.Background(), runID, run.Optimize.Metric, top); err != nil {
		logger.Error("save sweeps", "err", err)
		os.Exit(1)
	}

	fmt.Printf("Optimization Run ID: %d\n", runID)
	fmt.Printf("Candidates:          %d\n", len(candidates))
	fmt.Printf("TopN saved:          %d\n", topN)
	fmt.Printf("Metric:              %s\n", run.Optimize.Metric)
	fmt.Printf("Best score:          %.4f\n", best.MetricValue)
	fmt.Printf("Best return:         %.2f%%\n", best.Metrics.TotalReturnPct)
	fmt.Printf("Best sharpe:         %.2f\n", best.Metrics.SharpeRatio)
	fmt.Printf("Best drawdown:       %.2f%%\n", best.Metrics.MaxDrawdownPct)
	fmt.Printf("Best params:         %v\n", best.Params)
}

func filterBars(bars []domain.Bar, from, to time.Time) []domain.Bar {
	var out []domain.Bar
	for _, b := range bars {
		if !b.OpenTime.Before(from) && !b.OpenTime.After(to) {
			out = append(out, b)
		}
	}
	return out
}

func getFloat(m map[string]any, key string, def float64) float64 {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return def
	}
}

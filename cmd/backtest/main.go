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
	"github.com/AinurKadirov/crypto-parser/internal/storage/parquet"
	"github.com/AinurKadirov/crypto-parser/internal/storage/sqlite"
	"github.com/AinurKadirov/crypto-parser/internal/strategy"
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
		logger.Error("no data", "hint", "make fetch-btc")
		os.Exit(1)
	}

	from, _ := time.Parse("2006-01-02", run.Run.DateFrom)
	to, _ := time.Parse("2006-01-02", run.Run.DateTo)
	to = to.Add(24*time.Hour - time.Millisecond)
	bars = filterBars(bars, from, to)

	reg := strategy.NewRegistry()
	strat, err := reg.Create(run.Run.Strategy, bars, stratCfg, run.Params)
	if err != nil {
		logger.Error("create strategy", "err", err)
		os.Exit(1)
	}

	risk := backtest.RiskConfig{
		InitialBalance: getFloat(stratCfg.Backtest, "initial_balance", 10000),
		CommissionPct:  stratCfg.Risk["commission_pct"],
		SlippagePct:    stratCfg.Risk["slippage_pct"],
		StopLossPct:    stratCfg.Risk["stop_loss_pct"],
		TakeProfitPct:  stratCfg.Risk["take_profit_pct"],
		MaxPositionPct: stratCfg.Risk["max_position_pct"],
	}

	result := backtest.NewEngine(risk).Run(bars, strat)

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

	runName := run.Run.Name
	if runName == "" {
		runName = fmt.Sprintf("%s_%s_%s", run.Run.Symbol, run.Run.Timeframe, run.Run.Strategy)
	}

	runID, err := db.SaveRunResult(context.Background(), sqlite.RunInput{
		Name:      runName,
		Strategy:  run.Run.Strategy,
		Symbol:    run.Run.Symbol,
		Exchange:  run.Run.Exchange,
		Timeframe: run.Run.Timeframe,
		DateFrom:  run.Run.DateFrom,
		DateTo:    run.Run.DateTo,
	}, result.Metrics, run.Params)
	if err != nil {
		logger.Error("save result", "err", err)
		os.Exit(1)
	}

	fmt.Printf("Strategy: %s\n", strat.Name())
	fmt.Printf("Symbol:   %s %s\n", run.Run.Symbol, run.Run.Timeframe)
	fmt.Printf("Bars:     %d\n", len(bars))
	fmt.Printf("Run ID:   %d\n", runID)
	fmt.Printf("Return:   %.2f%%\n", result.Metrics.TotalReturnPct)
	fmt.Printf("Sharpe:   %.2f\n", result.Metrics.SharpeRatio)
	fmt.Printf("MaxDD:    %.2f%%\n", result.Metrics.MaxDrawdownPct)
	fmt.Printf("Trades:   %d\n", result.Metrics.TotalTrades)
	fmt.Printf("WinRate:  %.2f%%\n", result.Metrics.WinRatePct)
	fmt.Printf("PF:       %.2f\n", result.Metrics.ProfitFactor)
	fmt.Printf("Final:    %.2f USDT\n", result.FinalBalance)
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

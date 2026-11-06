package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/config"
	"github.com/AinurKadirov/crypto-parser/internal/domain"
	"github.com/AinurKadirov/crypto-parser/internal/exchange/binance"
	"github.com/AinurKadirov/crypto-parser/internal/storage/parquet"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var (
		symbol    = flag.String("symbol", "", "Trading pair e.g. BTCUSDT")
		timeframe = flag.String("timeframe", "", "Candle interval e.g. 1h")
		fromStr   = flag.String("from", "", "Start date YYYY-MM-DD")
		toStr     = flag.String("to", "", "End date YYYY-MM-DD")
		runCfg    = flag.String("run", "", "Run config path e.g. configs/runs/btc_1h_ema.yaml")
	)
	flag.Parse()

	loader := config.NewLoader("./configs")
	appCfg, err := loader.LoadApp()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}
	configDir := config.ResolveConfigDir(appCfg)
	loader = config.NewLoader(configDir)

	exchangeName := "binance"
	if *runCfg != "" {
		run, err := loader.LoadRun(*runCfg)
		if err != nil {
			logger.Error("load run", "err", err)
			os.Exit(1)
		}
		*symbol = run.Run.Symbol
		*timeframe = run.Run.Timeframe
		*fromStr = run.Run.DateFrom
		*toStr = run.Run.DateTo
		exchangeName = run.Run.Exchange
	}

	if *symbol == "" || *timeframe == "" || *fromStr == "" || *toStr == "" {
		fmt.Fprintln(os.Stderr, "Usage: fetch --symbol BTCUSDT --timeframe 1h --from 2024-01-01 --to 2024-06-01")
		fmt.Fprintln(os.Stderr, "   or: fetch --run configs/runs/btc_1h_ema.yaml")
		os.Exit(2)
	}

	from, err := time.Parse("2006-01-02", *fromStr)
	if err != nil {
		logger.Error("parse from date", "err", err)
		os.Exit(1)
	}
	to, err := time.Parse("2006-01-02", *toStr)
	if err != nil {
		logger.Error("parse to date", "err", err)
		os.Exit(1)
	}
	to = to.Add(24*time.Hour - time.Millisecond)

	client := binance.NewClient(appCfg.Fetch.BinanceBaseURL, appCfg.Fetch.RateLimitMS)
	ctx := context.Background()

	logger.Info("fetching klines",
		"symbol", *symbol,
		"timeframe", *timeframe,
		"from", from.Format("2006-01-02"),
		"to", to.Format("2006-01-02"),
	)

	bars, err := client.FetchKlines(ctx, *symbol, *timeframe, from, to)
	if err != nil {
		logger.Error("fetch failed", "err", err)
		os.Exit(1)
	}

	dataDir := config.ResolveDataDir(appCfg)
	repo := parquet.NewCandleRepo(dataDir)
	key := domain.SymbolKey{
		Exchange:  exchangeName,
		Symbol:    *symbol,
		Timeframe: *timeframe,
	}

	if err := repo.Save(ctx, key, bars); err != nil {
		logger.Error("save parquet", "err", err)
		os.Exit(1)
	}

	logger.Info("done",
		"bars", len(bars),
		"path", fmt.Sprintf("%s/%s/%s/%s.parquet", dataDir, key.Exchange, key.Symbol, key.Timeframe),
	)
}

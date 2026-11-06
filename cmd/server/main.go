package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"github.com/AinurKadirov/crypto-parser/internal/config"
	"github.com/AinurKadirov/crypto-parser/internal/indicator"
	"github.com/AinurKadirov/crypto-parser/internal/server"
	"github.com/AinurKadirov/crypto-parser/internal/storage/parquet"
	"github.com/AinurKadirov/crypto-parser/internal/storage/sqlite"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	loader := config.NewLoader("./configs")
	appCfg, err := loader.LoadApp()
	if err != nil {
		logger.Error("load app config", "err", err)
		os.Exit(1)
	}

	configDir := config.ResolveConfigDir(appCfg)
	loader = config.NewLoader(configDir)

	dsn := appCfg.Database.DSN
	if env := os.Getenv("CRYPTO_PARSER_DB_DSN"); env != "" {
		dsn = strings.TrimPrefix(env, "sqlite:")
	}
	db, err := sqlite.Open(dsn)
	if err != nil {
		logger.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	dataDir := config.ResolveDataDir(appCfg)
	candles := parquet.NewCandleRepo(dataDir)
	indicators := indicator.NewService()

	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.Error("embed static", "err", err)
		os.Exit(1)
	}

	srv := server.New(appCfg, loader, candles, indicators, db, staticSub, logger)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/config"
	"github.com/AinurKadirov/crypto-parser/internal/domain"
	"github.com/AinurKadirov/crypto-parser/internal/indicator"
	"github.com/AinurKadirov/crypto-parser/internal/storage"
	"github.com/AinurKadirov/crypto-parser/internal/storage/sqlite"
)

// Server serves the dashboard API and static UI.
type Server struct {
	cfg        *config.App
	loader     *config.Loader
	candles    storage.CandleRepository
	indicators *indicator.Service
	db         *sqlite.DB
	staticFS   fs.FS
	logger     *slog.Logger
}

// New creates an HTTP server with injected dependencies (DIP).
func New(cfg *config.App, loader *config.Loader, candles storage.CandleRepository, indicators *indicator.Service, db *sqlite.DB, staticFS fs.FS, logger *slog.Logger) *Server {
	return &Server{
		cfg:        cfg,
		loader:     loader,
		candles:    candles,
		indicators: indicators,
		db:         db,
		staticFS:   staticFS,
		logger:     logger,
	}
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/overview", s.handleOverview)
	mux.HandleFunc("GET /api/configs", s.handleConfigs)
	mux.HandleFunc("GET /api/data", s.handleDataInventory)
	mux.HandleFunc("GET /api/runs", s.handleRuns)
	mux.HandleFunc("GET /api/sweeps", s.handleSweeps)
	mux.HandleFunc("GET /api/indicators", s.handleIndicators)

	fileServer := http.FileServer(http.FS(s.staticFS))
	mux.Handle("/", fileServer)

	return mux
}

// ListenAndServe starts the HTTP server on the configured port.
func (s *Server) ListenAndServe() error {
	port := s.cfg.UI.Port
	if p := os.Getenv("CRYPTO_PARSER_UI_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	addr := ":" + strconv.Itoa(port)
	s.logger.Info("UI server starting", "addr", "http://localhost"+addr)
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := "ok"
	dbStatus := "ok"
	if err := s.db.Ping(ctx); err != nil {
		dbStatus = "error"
		status = "degraded"
	}
	writeJSON(w, map[string]any{
		"status":   status,
		"database": dbStatus,
		"app":      s.cfg.App.Name,
		"time":     time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	configs, _ := s.loader.ListConfigs()
	dataDir := config.ResolveDataDir(s.cfg)
	inv, _ := s.candles.Inventory(ctx, dataDir)
	runs, _ := s.db.ListRuns(ctx, 5)

	writeJSON(w, map[string]any{
		"app": map[string]any{
			"name":    s.cfg.App.Name,
			"version": "0.5.0-sprint5",
		},
		"stats": map[string]any{
			"configs":    len(configs),
			"data_files": len(inv),
			"runs":       len(runs),
		},
		"stack": map[string]string{
			"language": "Go 1.22",
			"candles":  "Parquet",
			"results":  s.cfg.Database.Driver,
			"docs":     "docs/",
		},
	})
}

func (s *Server) handleConfigs(w http.ResponseWriter, r *http.Request) {
	files, err := s.loader.ListConfigs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	app, _ := s.loader.LoadApp()
	pool, _ := s.loader.LoadPool("major.yaml")
	strategy, _ := s.loader.LoadStrategy("ema_crossover")
	run, _ := s.loader.LoadRun("btc_1h_ema.yaml")

	writeJSON(w, map[string]any{
		"files": files,
		"details": map[string]any{
			"app":        app,
			"pool":       pool,
			"strategy":   strategy,
			"sample_run": run,
		},
	})
}

func (s *Server) handleDataInventory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dataDir := config.ResolveDataDir(s.cfg)
	inv, err := s.candles.Inventory(ctx, dataDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"inventory": inv, "data_dir": dataDir})
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.db.ListRunsWithMetrics(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"runs": runs})
}

func (s *Server) handleSweeps(w http.ResponseWriter, r *http.Request) {
	sweeps, err := s.db.ListSweeps(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"sweeps": sweeps})
}

func (s *Server) handleIndicators(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	symbol := r.URL.Query().Get("symbol")
	timeframe := r.URL.Query().Get("timeframe")
	exchange := r.URL.Query().Get("exchange")
	if exchange == "" {
		exchange = "binance"
	}
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	if timeframe == "" {
		timeframe = "1h"
	}

	key := domain.SymbolKey{Exchange: exchange, Symbol: symbol, Timeframe: timeframe}
	bars, err := s.candles.Load(ctx, key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(bars) == 0 {
		writeError(w, http.StatusNotFound, fmt.Errorf("no data for %s %s %s — run: make fetch-btc", exchange, symbol, timeframe))
		return
	}

	preview := s.indicators.PreviewFromBars(ctx, symbol, timeframe, bars, 30)
	writeJSON(w, preview)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

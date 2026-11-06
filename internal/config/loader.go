package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// App holds global application settings.
type App struct {
	App struct {
		Name     string `yaml:"name"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"app"`
	Paths struct {
		DataDir    string `yaml:"data_dir"`
		ResultsDir string `yaml:"results_dir"`
		ConfigDir  string `yaml:"config_dir"`
	} `yaml:"paths"`
	Database struct {
		Driver string `yaml:"driver"`
		DSN    string `yaml:"dsn"`
	} `yaml:"database"`
	UI struct {
		Port    int  `yaml:"port"`
		Enabled bool `yaml:"enabled"`
	} `yaml:"ui"`
	Fetch struct {
		RateLimitMS    int    `yaml:"rate_limit_ms"`
		RetryCount     int    `yaml:"retry_count"`
		BinanceBaseURL string `yaml:"binance_base_url"`
	} `yaml:"fetch"`
}

// Pool defines a group of symbols for batch runs.
type Pool struct {
	Pool struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Exchange    string `yaml:"exchange"`
	} `yaml:"pool"`
	Symbols    []string `yaml:"symbols"`
	Timeframes []string `yaml:"timeframes"`
}

// ParamDef describes a single strategy parameter.
type ParamDef struct {
	Type    string    `yaml:"type"`
	Default any       `yaml:"default"`
	Range   []float64 `yaml:"range"`
	Step    float64   `yaml:"step"`
}

// Strategy defines a trading strategy configuration.
type Strategy struct {
	Strategy struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
	} `yaml:"strategy"`
	Parameters map[string]ParamDef `yaml:"parameters"`
	Filters    map[string]any      `yaml:"filters"`
	Risk       map[string]float64  `yaml:"risk"`
	Backtest   map[string]any      `yaml:"backtest"`
}

// Run defines a single backtest or optimization run.
type Run struct {
	Run struct {
		Name      string `yaml:"name"`
		Strategy  string `yaml:"strategy"`
		Symbol    string `yaml:"symbol"`
		Exchange  string `yaml:"exchange"`
		Timeframe string `yaml:"timeframe"`
		DateFrom  string `yaml:"date_from"`
		DateTo    string `yaml:"date_to"`
	} `yaml:"run"`
	Params    map[string]any `yaml:"params"`
	Optimize  OptimizeConfig `yaml:"optimize"`
}

// OptimizeConfig controls parameter search.
type OptimizeConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Method         string `yaml:"method"`
	Metric         string `yaml:"metric"`
	TopN           int    `yaml:"top_n"`
	MaxIterations  int    `yaml:"max_iterations"`
}

// ConfigFile is a lightweight descriptor for UI listing.
type ConfigFile struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Path     string `json:"path"`
}

// Loader reads YAML configs from disk (Single Responsibility).
type Loader struct {
	configDir string
}

// NewLoader creates a config loader for the given directory.
func NewLoader(configDir string) *Loader {
	return &Loader{configDir: configDir}
}

// LoadApp reads configs/app.yaml.
func (l *Loader) LoadApp() (*App, error) {
	return loadYAML[App](filepath.Join(l.configDir, "app.yaml"))
}

// LoadPool reads a pool config by filename (e.g. major.yaml).
func (l *Loader) LoadPool(filename string) (*Pool, error) {
	return loadYAML[Pool](filepath.Join(l.configDir, "pools", filename))
}

// LoadStrategy reads a strategy config by name (without .yaml).
func (l *Loader) LoadStrategy(name string) (*Strategy, error) {
	return loadYAML[Strategy](filepath.Join(l.configDir, "strategies", name+".yaml"))
}

// LoadRun reads a run config by path.
func (l *Loader) LoadRun(path string) (*Run, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(l.configDir, "runs", filepath.Base(path))
	}
	return loadYAML[Run](path)
}

// ListConfigs returns all YAML files grouped by category.
func (l *Loader) ListConfigs() ([]ConfigFile, error) {
	categories := map[string]string{
		"pools":      "pool",
		"strategies": "strategy",
		"runs":       "run",
	}
	var files []ConfigFile
	for dir, cat := range categories {
		entries, err := os.ReadDir(filepath.Join(l.configDir, dir))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			files = append(files, ConfigFile{
				Category: cat,
				Name:     strings.TrimSuffix(e.Name(), ".yaml"),
				Path:     filepath.Join("configs", dir, e.Name()),
			})
		}
	}
	return files, nil
}

func loadYAML[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var v T
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &v, nil
}

// ResolveConfigDir returns config dir from env or default.
func ResolveConfigDir(app *App) string {
	if dir := os.Getenv("CRYPTO_PARSER_CONFIG_DIR"); dir != "" {
		return dir
	}
	if app != nil && app.Paths.ConfigDir != "" {
		return app.Paths.ConfigDir
	}
	return "./configs"
}

// ResolveDataDir returns data directory path.
func ResolveDataDir(app *App) string {
	if dir := os.Getenv("CRYPTO_PARSER_DATA_DIR"); dir != "" {
		return dir
	}
	if app != nil && app.Paths.DataDir != "" {
		return app.Paths.DataDir
	}
	return "./data"
}

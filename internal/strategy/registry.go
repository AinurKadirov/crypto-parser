package strategy

import (
	"fmt"

	"github.com/AinurKadirov/crypto-parser/internal/config"
	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// Registry creates a prepared strategy by name (Factory pattern).
type Registry struct{}

// NewRegistry returns a strategy factory.
func NewRegistry() *Registry {
	return &Registry{}
}

// Create builds a strategy instance for the given bars and configuration.
func (r *Registry) Create(name string, bars []domain.Bar, stratCfg *config.Strategy, runParams map[string]any) (Strategy, error) {
	switch name {
	case "ema_crossover":
		params := ParamsFromConfig(stratCfg, runParams)
		return NewEMACrossover(bars, params)
	case "buy_and_hold":
		return BuyAndHold{}, nil
	default:
		return nil, fmt.Errorf("unknown strategy: %s", name)
	}
}

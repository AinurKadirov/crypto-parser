package storage

import (
	"context"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// CandleRepository persists OHLCV bars (Repository pattern).
type CandleRepository interface {
	Save(ctx context.Context, key domain.SymbolKey, bars []domain.Bar) error
	Load(ctx context.Context, key domain.SymbolKey) ([]domain.Bar, error)
	Inventory(ctx context.Context, dataDir string) ([]domain.DataInventory, error)
}

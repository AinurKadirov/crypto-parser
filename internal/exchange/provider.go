package exchange

import (
	"context"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// MarketDataProvider fetches OHLCV data from an exchange (Adapter pattern).
type MarketDataProvider interface {
	Name() string
	FetchKlines(ctx context.Context, symbol, timeframe string, from, to time.Time) ([]domain.Bar, error)
}

// TimeframeToBinance maps YAML timeframe to Binance interval string.
func TimeframeToBinance(tf string) string {
	return tf
}

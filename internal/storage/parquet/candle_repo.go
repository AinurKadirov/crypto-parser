package parquet

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// row is the Parquet schema for a candlestick.
type row struct {
	OpenTimeMs  int64   `parquet:"open_time_ms"`
	Open        float64 `parquet:"open"`
	High        float64 `parquet:"high"`
	Low         float64 `parquet:"low"`
	Close       float64 `parquet:"close"`
	Volume      float64 `parquet:"volume"`
	CloseTimeMs int64   `parquet:"close_time_ms"`
}

// CandleRepo stores candles as Parquet files on disk.
type CandleRepo struct {
	baseDir string
}

// NewCandleRepo creates a Parquet-backed candle repository.
func NewCandleRepo(baseDir string) *CandleRepo {
	return &CandleRepo{baseDir: baseDir}
}

// Save writes bars to data/{exchange}/{symbol}/{timeframe}.parquet (merge by timestamp).
func (r *CandleRepo) Save(ctx context.Context, key domain.SymbolKey, bars []domain.Bar) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(bars) == 0 {
		return nil
	}

	existing, _ := r.Load(ctx, key)
	merged := mergeBars(existing, bars)
	rows := barsToRows(merged)

	path := r.filePath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create parquet: %w", err)
	}
	defer f.Close()

	w := parquet.NewGenericWriter[row](f)
	if _, err := w.Write(rows); err != nil {
		return fmt.Errorf("write parquet: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close parquet writer: %w", err)
	}
	return nil
}

// Load reads all bars for the given symbol key.
func (r *CandleRepo) Load(ctx context.Context, key domain.SymbolKey) ([]domain.Bar, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path := r.filePath(key)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open parquet: %w", err)
	}
	defer f.Close()

	rdr := parquet.NewGenericReader[row](f)
	defer rdr.Close()

	buf := make([]row, 1024)
	var rows []row
	for {
		n, err := rdr.Read(buf)
		if n > 0 {
			rows = append(rows, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read parquet: %w", err)
		}
	}

	return rowsToBars(rows), nil
}

// Inventory scans data directory and returns metadata for each Parquet file.
func (r *CandleRepo) Inventory(ctx context.Context, dataDir string) ([]domain.DataInventory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dataDir == "" {
		dataDir = r.baseDir
	}

	var inv []domain.DataInventory
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".parquet") {
			return nil
		}
		key, ok := r.keyFromPath(dataDir, path)
		if !ok {
			return nil
		}
		bars, err := r.Load(ctx, key)
		if err != nil {
			return err
		}
		item := domain.DataInventory{
			Key:       key,
			BarCount:  len(bars),
			FilePath:  path,
			FileSizeB: info.Size(),
		}
		if len(bars) > 0 {
			item.FirstBar = bars[0].OpenTime
			item.LastBar = bars[len(bars)-1].OpenTime
		}
		inv = append(inv, item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(inv, func(i, j int) bool {
		return inv[i].Key.Symbol < inv[j].Key.Symbol
	})
	return inv, nil
}

func (r *CandleRepo) filePath(key domain.SymbolKey) string {
	return filepath.Join(r.baseDir, key.Exchange, key.Symbol, key.Timeframe+".parquet")
}

func (r *CandleRepo) keyFromPath(baseDir, path string) (domain.SymbolKey, bool) {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return domain.SymbolKey{}, false
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) < 3 {
		return domain.SymbolKey{}, false
	}
	tf := strings.TrimSuffix(parts[len(parts)-1], ".parquet")
	return domain.SymbolKey{
		Exchange:  parts[0],
		Symbol:    parts[1],
		Timeframe: tf,
	}, true
}

func mergeBars(existing, incoming []domain.Bar) []domain.Bar {
	byTime := make(map[int64]domain.Bar, len(existing)+len(incoming))
	for _, b := range existing {
		byTime[b.OpenTime.UnixMilli()] = b
	}
	for _, b := range incoming {
		byTime[b.OpenTime.UnixMilli()] = b
	}
	keys := make([]int64, 0, len(byTime))
	for k := range byTime {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	out := make([]domain.Bar, len(keys))
	for i, k := range keys {
		out[i] = byTime[k]
	}
	return out
}

func barsToRows(bars []domain.Bar) []row {
	rows := make([]row, len(bars))
	for i, b := range bars {
		rows[i] = row{
			OpenTimeMs:  b.OpenTime.UnixMilli(),
			Open:        b.Open,
			High:        b.High,
			Low:         b.Low,
			Close:       b.Close,
			Volume:      b.Volume,
			CloseTimeMs: b.CloseTime.UnixMilli(),
		}
	}
	return rows
}

func rowsToBars(rows []row) []domain.Bar {
	bars := make([]domain.Bar, len(rows))
	for i, r := range rows {
		bars[i] = domain.Bar{
			OpenTime:  time.UnixMilli(r.OpenTimeMs),
			Open:      r.Open,
			High:      r.High,
			Low:       r.Low,
			Close:     r.Close,
			Volume:    r.Volume,
			CloseTime: time.UnixMilli(r.CloseTimeMs),
		}
	}
	return bars
}

package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// Client implements exchange.MarketDataProvider for Binance Spot.
type Client struct {
	baseURL    string
	httpClient *http.Client
	rateLimit  time.Duration
}

// NewClient creates a Binance API client.
func NewClient(baseURL string, rateLimitMS int) *Client {
	if baseURL == "" {
		baseURL = "https://api.binance.com"
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: time.Duration(rateLimitMS) * time.Millisecond,
	}
}

// Name returns the exchange identifier.
func (c *Client) Name() string {
	return "binance"
}

// FetchKlines downloads historical candles with pagination (max 1000 per request).
func (c *Client) FetchKlines(ctx context.Context, symbol, timeframe string, from, to time.Time) ([]domain.Bar, error) {
	const limit = 1000
	startMs := from.UnixMilli()
	endMs := to.UnixMilli()

	var all []domain.Bar
	for startMs < endMs {
		batch, err := c.fetchBatch(ctx, symbol, timeframe, startMs, endMs, limit)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		last := batch[len(batch)-1]
		nextStart := last.CloseTime.UnixMilli() + 1
		if nextStart <= startMs {
			break
		}
		startMs = nextStart

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.rateLimit):
		}
	}
	return all, nil
}

func (c *Client) fetchBatch(ctx context.Context, symbol, timeframe string, startMs, endMs int64, limit int) ([]domain.Bar, error) {
	u, err := url.Parse(c.baseURL + "/api/v3/klines")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("symbol", symbol)
	q.Set("interval", timeframe)
	q.Set("startTime", strconv.FormatInt(startMs, 10))
	q.Set("endTime", strconv.FormatInt(endMs, 10))
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance API %d: %s", resp.StatusCode, string(body))
	}

	var raw [][]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse klines: %w", err)
	}

	bars := make([]domain.Bar, 0, len(raw))
	for _, row := range raw {
		bar, err := parseKlineRow(row)
		if err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}
	return bars, nil
}

func parseKlineRow(row []json.RawMessage) (domain.Bar, error) {
	if len(row) < 7 {
		return domain.Bar{}, fmt.Errorf("invalid kline row length %d", len(row))
	}
	openMs, err := parseJSONInt64(row[0])
	if err != nil {
		return domain.Bar{}, err
	}
	open, err := parseJSONFloat(row[1])
	if err != nil {
		return domain.Bar{}, err
	}
	high, err := parseJSONFloat(row[2])
	if err != nil {
		return domain.Bar{}, err
	}
	low, err := parseJSONFloat(row[3])
	if err != nil {
		return domain.Bar{}, err
	}
	closeP, err := parseJSONFloat(row[4])
	if err != nil {
		return domain.Bar{}, err
	}
	vol, err := parseJSONFloat(row[5])
	if err != nil {
		return domain.Bar{}, err
	}
	closeMs, err := parseJSONInt64(row[6])
	if err != nil {
		return domain.Bar{}, err
	}
	return domain.Bar{
		OpenTime:  time.UnixMilli(openMs),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     closeP,
		Volume:    vol,
		CloseTime: time.UnixMilli(closeMs),
	}, nil
}

func parseJSONInt64(raw json.RawMessage) (int64, error) {
	var v int64
	if err := json.Unmarshal(raw, &v); err == nil {
		return v, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

func parseJSONFloat(raw json.RawMessage) (float64, error) {
	var v float64
	if err := json.Unmarshal(raw, &v); err == nil {
		return v, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

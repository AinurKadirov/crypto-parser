package indicator

import (
	"math"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

// Series is a named indicator output aligned with input bars.
// Values before warmup are NaN.
type Series struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}

// Calculator computes a technical indicator from close prices.
type Calculator interface {
	Name() string
	Compute(closes []float64) Series
}

// BarCalculator uses full OHLCV bars when volume is required.
type BarCalculator interface {
	Name() string
	ComputeBars(bars []BarInput) Series
}

// BarInput is a minimal OHLCV view for indicators.
type BarInput struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// BarsFromDomain converts domain bars to indicator input.
func BarsFromDomain(bars []domain.Bar) []BarInput {
	out := make([]BarInput, len(bars))
	for i, b := range bars {
		out[i] = BarInput{
			Open: b.Open, High: b.High, Low: b.Low,
			Close: b.Close, Volume: b.Volume,
		}
	}
	return out
}

// ClosesFromBars extracts close prices.
func ClosesFromBars(bars []BarInput) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = b.Close
	}
	return out
}

// VolumesFromBars extracts volumes.
func VolumesFromBars(bars []BarInput) []float64 {
	out := make([]float64, len(bars))
	for i, b := range bars {
		out[i] = b.Volume
	}
	return out
}

func nanSeries(name string, n int) Series {
	v := make([]float64, n)
	for i := range v {
		v[i] = math.NaN()
	}
	return Series{Name: name, Values: v}
}

func isValid(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

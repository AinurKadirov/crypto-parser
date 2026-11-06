package indicator

import "fmt"

// MACDResult holds MACD line, signal, and histogram.
type MACDResult struct {
	MACD      Series
	Signal    Series
	Histogram Series
}

// MACD computes MACD(12,26,9) by default.
type MACD struct {
	Fast   int
	Slow   int
	Signal int
}

func (m MACD) Name() string {
	return fmt.Sprintf("MACD(%d,%d,%d)", m.fast(), m.slow(), m.signal())
}

func (m MACD) fast() int {
	if m.Fast <= 0 {
		return 12
	}
	return m.Fast
}

func (m MACD) slow() int {
	if m.Slow <= 0 {
		return 26
	}
	return m.Slow
}

func (m MACD) signal() int {
	if m.Signal <= 0 {
		return 9
	}
	return m.Signal
}

// Compute returns the MACD line only (implements Calculator).
func (m MACD) Compute(closes []float64) Series {
	return m.ComputeFull(closes).MACD
}

// ComputeFull returns MACD, signal, and histogram series.
func (m MACD) ComputeFull(closes []float64) MACDResult {
	n := len(closes)
	fast := EMA{Period: m.fast()}.Compute(closes)
	slow := EMA{Period: m.slow()}.Compute(closes)

	macdLine := nanSeries(m.Name(), n)
	for i := 0; i < n; i++ {
		if isValid(fast.Values[i]) && isValid(slow.Values[i]) {
			macdLine.Values[i] = fast.Values[i] - slow.Values[i]
		}
	}

	signalLine := EMA{Period: m.signal()}.ComputeAligned(macdLine.Values)
	hist := nanSeries("MACD_HIST", n)
	for i := 0; i < n; i++ {
		if isValid(macdLine.Values[i]) && isValid(signalLine.Values[i]) {
			hist.Values[i] = macdLine.Values[i] - signalLine.Values[i]
		}
	}

	return MACDResult{
		MACD:      macdLine,
		Signal:    signalLine,
		Histogram: hist,
	}
}

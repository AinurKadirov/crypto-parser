package indicator

import "fmt"

// RSI computes Relative Strength Index (Wilder's smoothing).
type RSI struct {
	Period int
}

func (r RSI) Name() string {
	return fmt.Sprintf("RSI(%d)", r.Period)
}

func (r RSI) Compute(closes []float64) Series {
	n := len(closes)
	period := r.Period
	if period <= 0 {
		period = 14
	}
	if n == 0 || n <= period {
		return nanSeries(r.Name(), n)
	}

	out := nanSeries(r.Name(), n)

	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		chg := closes[i] - closes[i-1]
		gainSum += max(chg, 0)
		lossSum += max(-chg, 0)
	}
	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)
	out.Values[period] = rsiValue(avgGain, avgLoss)

	for i := period + 1; i < n; i++ {
		chg := closes[i] - closes[i-1]
		gain := max(chg, 0)
		loss := max(-chg, 0)
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		out.Values[i] = rsiValue(avgGain, avgLoss)
	}
	return out
}

func rsiValue(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

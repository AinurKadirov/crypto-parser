package indicator

import "fmt"

// SMA computes Simple Moving Average.
type SMA struct {
	Period int
}

func (s SMA) Name() string {
	return fmt.Sprintf("SMA(%d)", s.Period)
}

func (s SMA) Compute(closes []float64) Series {
	n := len(closes)
	if s.Period <= 0 || n == 0 {
		return nanSeries(s.Name(), n)
	}
	out := nanSeries(s.Name(), n)
	var sum float64
	for i := 0; i < n; i++ {
		sum += closes[i]
		if i >= s.Period {
			sum -= closes[i-s.Period]
		}
		if i >= s.Period-1 {
			out.Values[i] = sum / float64(s.Period)
		}
	}
	return out
}

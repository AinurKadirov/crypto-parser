package indicator

import "fmt"

// EMA computes Exponential Moving Average.
// First valid value at index period-1 equals SMA(period).
type EMA struct {
	Period int
}

func (e EMA) Name() string {
	return fmt.Sprintf("EMA(%d)", e.Period)
}

func (e EMA) Compute(closes []float64) Series {
	n := len(closes)
	if e.Period <= 0 || n == 0 {
		return nanSeries(e.Name(), n)
	}
	out := nanSeries(e.Name(), n)
	k := 2.0 / float64(e.Period+1)

	var sum float64
	for i := 0; i < n; i++ {
		if i < e.Period-1 {
			sum += closes[i]
			continue
		}
		if i == e.Period-1 {
			sum += closes[i]
			out.Values[i] = sum / float64(e.Period)
			continue
		}
		prev := out.Values[i-1]
		out.Values[i] = closes[i]*k + prev*(1-k)
	}
	return out
}

// ComputeAligned calculates EMA on a series that may contain leading NaN gaps.
// Output indices align with input; warmup NaNs are preserved.
func (e EMA) ComputeAligned(values []float64) Series {
	n := len(values)
	if e.Period <= 0 || n == 0 {
		return nanSeries(e.Name(), n)
	}
	out := nanSeries(e.Name(), n)
	k := 2.0 / float64(e.Period+1)

	var sum float64
	valid := 0
	started := false

	for i, v := range values {
		if !isValid(v) {
			continue
		}
		if !started {
			valid++
			sum += v
			if valid < e.Period {
				continue
			}
			out.Values[i] = sum / float64(e.Period)
			started = true
			continue
		}
		prev := out.Values[i-1]
		if !isValid(prev) {
			for j := i - 1; j >= 0; j-- {
				if isValid(out.Values[j]) {
					prev = out.Values[j]
					break
				}
			}
		}
		out.Values[i] = v*k + prev*(1-k)
	}
	return out
}

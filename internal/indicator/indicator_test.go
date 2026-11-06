package indicator

import (
	"math"
	"testing"
)

func TestSMA(t *testing.T) {
	closes := []float64{1, 2, 3, 4, 5}
	s := SMA{Period: 3}.Compute(closes)
	want := []float64{math.NaN(), math.NaN(), 2, 3, 4}
	for i, w := range want {
		if math.IsNaN(w) {
			if !math.IsNaN(s.Values[i]) {
				t.Fatalf("index %d: want NaN got %v", i, s.Values[i])
			}
			continue
		}
		if math.Abs(s.Values[i]-w) > 1e-9 {
			t.Fatalf("index %d: want %v got %v", i, w, s.Values[i])
		}
	}
}

func TestEMA_firstValueIsSMA(t *testing.T) {
	closes := []float64{22, 22.27, 22.19, 22.08, 22.17}
	ema := EMA{Period: 3}.Compute(closes)
	sma := SMA{Period: 3}.Compute(closes)
	if math.Abs(ema.Values[2]-sma.Values[2]) > 1e-9 {
		t.Fatalf("EMA[2]=%v SMA[2]=%v", ema.Values[2], sma.Values[2])
	}
}

func TestEMA_knownSequence(t *testing.T) {
	closes := []float64{22, 22.27, 22.19, 22.08, 22.17}
	ema := EMA{Period: 3}.Compute(closes)
	// k=0.5; EMA[2]=SMA=22.1533; EMA[3]=22.08*0.5+22.1533*0.5=22.1167
	if math.Abs(ema.Values[3]-22.1167) > 0.001 {
		t.Errorf("EMA[3]=%v want ~22.1167", ema.Values[3])
	}
}

func TestRSI_range(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = 100 + float64(i%5)
	}
	rsi := RSI{Period: 14}.Compute(closes)
	for i := 14; i < len(closes); i++ {
		v := rsi.Values[i]
		if v < 0 || v > 100 {
			t.Fatalf("RSI out of range at %d: %v", i, v)
		}
	}
}

func TestMACD_lengths(t *testing.T) {
	closes := make([]float64, 50)
	for i := range closes {
		closes[i] = 100 + float64(i)*0.5
	}
	res := MACD{}.ComputeFull(closes)
	if len(res.MACD.Values) != len(closes) {
		t.Fatal("MACD length mismatch")
	}
}

func TestVolumeSMA(t *testing.T) {
	bars := []BarInput{
		{Volume: 100}, {Volume: 200}, {Volume: 300},
	}
	v := VolumeSMA{Period: 2}.ComputeBars(bars)
	if math.Abs(v.Values[1]-150) > 1e-9 {
		t.Fatalf("got %v want 150", v.Values[1])
	}
	if math.Abs(v.Values[2]-250) > 1e-9 {
		t.Fatalf("got %v want 250", v.Values[2])
	}
}

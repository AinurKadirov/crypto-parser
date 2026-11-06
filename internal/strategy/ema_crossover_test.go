package strategy

import (
	"testing"
	"time"

	"github.com/AinurKadirov/crypto-parser/internal/domain"
)

func TestEMACrossover_bullishCross(t *testing.T) {
	closes := make([]float64, 40)
	for i := 0; i < 20; i++ {
		closes[i] = 100
	}
	for i := 20; i < 40; i++ {
		closes[i] = 100 + float64(i-19)*3
	}

	bars := make([]domain.Bar, len(closes))
	for i, c := range closes {
		bars[i] = domain.Bar{
			OpenTime: time.Now().Add(time.Duration(i) * time.Hour),
			Close:    c,
			High:     c + 0.5,
			Low:      c - 0.5,
			Volume:   5000,
		}
	}

	s, err := NewEMACrossover(bars, EMACrossoverParams{
		EMAFast: 5, EMASlow: 15, VolumeMultiplier: 0.5, MaxPositionPct: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	gotBuy := false
	for i := range bars {
		sig := s.OnBar(bars[i], i, &Context{BarCount: len(bars)})
		if sig.Action == domain.ActionBuy {
			gotBuy = true
			break
		}
	}
	if !gotBuy {
		t.Fatal("expected at least one buy signal after uptrend")
	}
}

func TestEMACrossover_rejectsFastGteSlow(t *testing.T) {
	_, err := NewEMACrossover(nil, EMACrossoverParams{EMAFast: 26, EMASlow: 12})
	if err == nil {
		t.Fatal("expected error when ema_fast >= ema_slow")
	}
}

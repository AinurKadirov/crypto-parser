package binance

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseKlineRow(t *testing.T) {
	raw := [][]json.RawMessage{
		{
			json.RawMessage(`1704067200000`),
			json.RawMessage(`"42000.50"`),
			json.RawMessage(`"42500.00"`),
			json.RawMessage(`"41800.00"`),
			json.RawMessage(`"42200.00"`),
			json.RawMessage(`"1234.56"`),
			json.RawMessage(`1704070799999`),
		},
	}

	bar, err := parseKlineRow(raw[0])
	if err != nil {
		t.Fatal(err)
	}
	if bar.Open != 42000.50 {
		t.Errorf("open = %v", bar.Open)
	}
	if bar.Close != 42200.00 {
		t.Errorf("close = %v", bar.Close)
	}
	if !bar.OpenTime.Equal(time.UnixMilli(1704067200000)) {
		t.Errorf("open time = %v", bar.OpenTime)
	}
}

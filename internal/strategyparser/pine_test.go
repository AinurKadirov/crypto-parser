package strategyparser

import "testing"

func TestParsePineInputs_MatSnippet(t *testing.T) {
	src := `
fastLength   = input.int(21, "[MA] Fast MA Length", minval=1)
fastMAType   = input.string("EMA", "Fast MA Type", options=["SMA", "EMA", "KAMA"])
volatilityThreshold = input.float(1.0, "Volatility Threshold (%)", minval=0.1, step=0.1) / 100
useTrendFilter      = input.bool(true, "[Filter Settings] Use 15m Trend Filter")
`

	params, err := ParsePineInputs(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(params) != 4 {
		t.Fatalf("want 4 params, got %d", len(params))
	}

	if params[0].Name != "fastLength" || params[0].Type != "int" {
		t.Fatalf("unexpected first param: %+v", params[0])
	}
	if params[1].Type != "string" || len(params[1].Options) != 3 {
		t.Fatalf("string options not parsed: %+v", params[1])
	}
	if params[2].Type != "float" || params[2].Step == nil || *params[2].Step != 0.1 {
		t.Fatalf("float step not parsed: %+v", params[2])
	}
	if params[3].Type != "bool" || params[3].Default != true {
		t.Fatalf("bool default not parsed: %+v", params[3])
	}
}

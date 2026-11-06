package indicator

import "fmt"

// VolumeSMA computes simple moving average of volume.
type VolumeSMA struct {
	Period int
}

func (v VolumeSMA) Name() string {
	return fmt.Sprintf("VolumeSMA(%d)", v.Period)
}

func (v VolumeSMA) ComputeBars(bars []BarInput) Series {
	return SMA{Period: v.Period}.Compute(VolumesFromBars(bars))
}

// Compute satisfies Calculator using volume from bars passed as closes workaround — prefer ComputeBars.
func (v VolumeSMA) Compute(closes []float64) Series {
	return SMA{Period: v.Period}.Compute(closes)
}

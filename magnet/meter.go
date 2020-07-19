package magnet

import (
	"log"
	"strings"
)

// Power returns the current power measurement in Watts
func (f Magnet) Power() float64 {
	if f.stop == 0 {
		return 0
	}
	return 0.9655 * 11.229 * 1000 / f.stop.Hours()
}

// Print screen output
func (f Magnet) Print() {
	log.Printf("%10v %v %8.1f %10.3f\n", f.Name, f.count, f.Power(), f.Meter)
}

// InfluxMeasurement …
func (f Magnet) InfluxMeasurement() string {
	return "gas"
}

// InfluxFields …
func (f Magnet) InfluxFields() map[string]interface{} {
	return map[string]interface{}{
		"value": f.Meter,
	}
}

// InfluxTags …
func (f Magnet) InfluxTags() map[string]string {
	return map[string]string{
		"meter": strings.ToLower(f.Name),
	}
}

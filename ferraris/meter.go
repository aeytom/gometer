package ferraris

import (
	"log"
	"strings"
)

// Power returns the current power measurement in Watts
func (f Ferraris) Power() float64 {
	if f.stop == 0 {
		return 0
	}
	return (1000 / float64(f.RotationsPerKiloWattHour)) / f.stop.Hours()
}

// Print screen output
func (f Ferraris) Print() {
	log.Printf("%10v %2v %4v %7.1f %10.3f\n", f.Name, f.BcmPin, f.count, f.Power(), f.Meter)
}

// InfluxMeasurement …
func (f Ferraris) InfluxMeasurement() string {
	return "meter"
}

// InfluxFields …
func (f Ferraris) InfluxFields() map[string]interface{} {
	return map[string]interface{}{
		"value":   f.Meter,
		"wattage": f.Power(),
	}
}

// InfluxTags …
func (f Ferraris) InfluxTags() map[string]string {
	return map[string]string{
		"meter": strings.ToLower(f.Name),
	}
}

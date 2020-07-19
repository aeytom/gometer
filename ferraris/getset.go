package ferraris

import "strings"

// ID …
func (f Ferraris) ID() string {
	return strings.ToLower(f.Name)
}

// Get …
func (f Ferraris) Get() float32 {
	return f.Meter
}

// Label …
func (f Ferraris) Label() string {
	return f.label
}

// Set …
func (f *Ferraris) Set(value float32) {
	f.ResetMeter(value)
}

// Unit …
func (f Ferraris) Unit() string {
	return "kWh"
}

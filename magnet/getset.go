package magnet

import "strings"

// ID …
func (f Magnet) ID() string {
	return strings.ToLower(f.Name)
}

// Get …
func (f Magnet) Get() float32 {
	return f.Meter
}

// Label …
func (f Magnet) Label() string {
	return f.label
}

// Set …
func (f *Magnet) Set(value float32) {
	f.ResetMeter(value)
}

// Unit …
func (f Magnet) Unit() string {
	return "m³"
}

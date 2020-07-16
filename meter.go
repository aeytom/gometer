package main

// Meter is base for all meters
type Meter interface {
	Power() float64
	Print()
	InfluxMeasurement() string
	InfluxFields() map[string]interface{}
	InfluxTags() map[string]string
}

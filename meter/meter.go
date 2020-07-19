package meter

// Meter is base for all meters
type Meter interface {
	Power() float64
	Print()
	InfluxMeasurement() string
	InfluxFields() map[string]interface{}
	InfluxTags() map[string]string
}

// GetSet provide simple write interface
type GetSet interface {
	ID() string
	Get() float32
	Set(float32)
	Label() string
	Unit() string
}

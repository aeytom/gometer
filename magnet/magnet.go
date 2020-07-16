package magnet

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/aeytom/qmc5883l/qmc5883l"

	"os"
	"time"
)

const (
	// RangeAdjustmentFraction adjusts max/min values by Range / RangeAdjustmentFraction
	RangeAdjustmentFraction = 20
	// RangeTresholdFraction - treshold is range / RangeTresholdFraction
	RangeTresholdFraction = 5
)

// used settings
var (
	Verbose bool
)

// Magnet holds data for magnet sensor based meters
type Magnet struct {
	Name   string
	Meter  float32
	MinVal int16
	MaxVal int16
	//
	baseMeter float32
	count     int
	start     time.Time
	stop      time.Duration
	//
	sensor    *qmc5883l.QMC5883L
	expectLow bool
}

// New initializes a new Magnet struct
func New(name string) Magnet {
	f := Magnet{
		Name: name,
	}
	restore(&f)
	f.sensor = qmc5883l.New(qmc5883l.DfltBus, qmc5883l.DfltAddress)
	f.sensor.SetMode(qmc5883l.ModeCONT, qmc5883l.Odr200HZ, qmc5883l.Rng8G, qmc5883l.Osr512)
	f.start = time.Now()
	return f
}

// Close …
func (f *Magnet) Close() {
	f.Close()
}

// ResetMeter resets the meter to new base meter value
func (f *Magnet) ResetMeter(v float32) {
	if v > 0 {
		f.baseMeter = v
		f.Meter = v
		f.count = 0
		save(f)
	}
}

// EdgeDetected dies und das
func (f *Magnet) EdgeDetected() bool {
	now := time.Now()
	val, _, _, err := f.sensor.GetMagnetRaw()
	if err != nil {
		log.Print(err)
		return false
	}

	if f.MinVal > val {
		f.MinVal = val
	}
	if f.MaxVal < val {
		f.MaxVal = val
	}

	if Verbose {
		log.Printf("gas %6d < %6d < %6d --- %9.3f %v", f.MinVal, val, f.MaxVal, f.Meter, f.expectLow)
	}

	xrange := f.MaxVal - f.MinVal
	if xrange > 5000 {
		if f.expectLow {
			if val < (f.MinVal + xrange/RangeTresholdFraction) {
				f.expectLow = false
				f.MinVal += xrange / RangeAdjustmentFraction
				return false
			}
		} else {
			if val > (f.MaxVal - xrange/RangeTresholdFraction) {
				f.expectLow = true
				f.MaxVal -= xrange / RangeAdjustmentFraction
				f.stop = now.Sub(f.start)
				f.start = now
				f.count++
				f.Meter = f.baseMeter + float32(f.count)*0.001
				save(f)
				return true
			}
		}
	}
	return false
}

// Power returns the current power measurement in Watts
func (f *Magnet) Power() float64 {
	if f.stop == 0 {
		return 0
	}
	return 0.9655 * 11.229 * 1000 / f.stop.Hours()
}

// Print screen output
func (f *Magnet) Print() {
	log.Printf("%10v %v %8.1f %10.3f\n", f.Name, f.count, f.Power(), f.Meter)
}

// InfluxMeasurement …
func (f *Magnet) InfluxMeasurement() string {
	return "gas"
}

// InfluxFields …
func (f *Magnet) InfluxFields() map[string]interface{} {
	return map[string]interface{}{
		"value": f.Meter,
	}
}

// InfluxTags …
func (f *Magnet) InfluxTags() map[string]string {
	return map[string]string{
		"meter": strings.ToLower(f.Name),
	}
}

//
func restore(f *Magnet) {
	b, err := ioutil.ReadFile(f.Name + ".json")
	if err != nil {
		log.Println(err)
		return
	}

	err = json.Unmarshal(b, f)
	if err != nil {
		log.Fatal(err)
	}

	f.ResetMeter(f.Meter)
}

//
func save(f *Magnet) {
	fpath := f.Name + ".json"
	file, err := os.OpenFile(fpath+".new", os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	jstring, err := json.MarshalIndent(f, "", "  ")
	file.Write(jstring)
	file.Close()

	os.Rename(fpath+".new", fpath)
}

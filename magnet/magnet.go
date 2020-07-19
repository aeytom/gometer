package magnet

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/aeytom/gometer/parameters"

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
	label     string
	baseMeter float32
	count     int
	start     time.Time
	stop      time.Duration
	//
	sensor    *qmc5883l.QMC5883L
	expectLow bool
	dorCount  int
}

// NewMagnet initializes a new Magnet struct
func NewMagnet(name string, label string) Magnet {
	f := Magnet{
		Name:  name,
		label: label,
	}
	restore(&f)
	f.sensor = qmc5883l.New(qmc5883l.DfltBus, qmc5883l.DfltAddress)
	f.sensor.SetMode(qmc5883l.ModeCONT, qmc5883l.Odr200HZ, qmc5883l.Rng8G, qmc5883l.Osr512)
	f.start = time.Now()
	return f
}

// Close …
func (f *Magnet) Close() {
	f.sensor.Close()
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
	if *parameters.Testing {
		return false
	}

	now := time.Now()
	val, _, _, err := f.sensor.GetMagnetRaw()
	if err != nil {
		log.Print(err)
		f.dorCount++
		if f.dorCount > 50 {
			panic("Magnet sensor has to often no data ready")
		}
		return false
	}
	f.dorCount = 0

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
	if *parameters.Testing {
		if *parameters.Verbose {
			log.Printf("not save state - %s to %s := %v", f.Name, fpath, f.Meter)
		}
		return
	}
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

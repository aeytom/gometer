package ferraris

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/stianeikeland/go-rpio"
)

// Ferraris …
type Ferraris struct {
	Name                     string
	BcmPin                   int
	RotationsPerKiloWattHour int
	Meter                    float32
	//
	pin       rpio.Pin
	baseMeter float32
	count     int
	state     rpio.State
	start     time.Time
	stop      time.Duration
}

// New …
func New(name string, pin int, rpkwh int) Ferraris {

	if err := rpio.Open(); err != nil {
		log.Fatal(err)
	}

	f := Ferraris{
		Name:                     name,
		BcmPin:                   pin,
		RotationsPerKiloWattHour: rpkwh,
		pin:                      rpio.Pin(pin),
	}
	restore(&f)
	f.pin.Input()
	f.pin.PullUp()
	f.state = f.pin.Read()
	f.start = time.Now()
	return f
}

// Close rpio
func (f *Ferraris) Close() {
	rpio.Close()
}

// ResetMeter resets the meter to new base meter value
func (f *Ferraris) ResetMeter(v float32) {
	if v > 0 {
		f.Meter = v
		f.baseMeter = v
		f.count = 0
		save(f)
	}
}

// Get return the current input state
func (f *Ferraris) Get() rpio.State {
	return f.pin.Read()
}

// Power returns the current power measurement in Watts
func (f *Ferraris) Power() float64 {
	if f.stop == 0 {
		return 0
	}
	return (1000 / float64(f.RotationsPerKiloWattHour)) / f.stop.Hours()
}

// EdgeDetected checks for a complete cycle
func (f *Ferraris) EdgeDetected() bool {
	now := time.Now()
	s := f.pin.Read()
	if f.state != s {
		f.state = s
		if s == rpio.High {
			f.stop = now.Sub(f.start)
			f.start = now
			f.count++
			f.Meter = f.baseMeter + float32(f.count)/float32(f.RotationsPerKiloWattHour)
			save(f)
			return true
		}
	}
	return false
}

// Print screen output
func (f *Ferraris) Print() {
	log.Printf("%10v %2v %4v %7.1f %10.3f\n", f.Name, f.BcmPin, f.count, f.Power(), f.Meter)
}

// InfluxMeasurement …
func (f *Ferraris) InfluxMeasurement() string {
	return "meter"
}

// InfluxFields …
func (f *Ferraris) InfluxFields() map[string]interface{} {
	return map[string]interface{}{
		"value":   f.Meter,
		"wattage": f.Power(),
	}
}

// InfluxTags …
func (f *Ferraris) InfluxTags() map[string]string {
	return map[string]string{
		"meter": strings.ToLower(f.Name),
	}
}

//
func restore(f *Ferraris) {
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
func save(f *Ferraris) {
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

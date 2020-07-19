package ferraris

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/aeytom/gometer/parameters"
	"github.com/stianeikeland/go-rpio"
)

// Ferraris …
type Ferraris struct {
	Name                     string
	BcmPin                   int
	RotationsPerKiloWattHour int
	Meter                    float32
	//
	label     string
	pin       rpio.Pin
	baseMeter float32
	count     int
	state     rpio.State
	start     time.Time
	stop      time.Duration
}

var rpioOpened bool

func init() {
	if err := rpio.Open(); err != nil {
		log.Fatal(err)
	}
	rpioOpened = true
	fmt.Println("init rpio")
}

// NewFerraris …
func NewFerraris(name string, pin int, rpkwh int, label string) Ferraris {

	f := Ferraris{
		Name:                     name,
		BcmPin:                   pin,
		RotationsPerKiloWattHour: rpkwh,
		pin:                      rpio.Pin(pin),
		label:                    label,
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
	if rpioOpened {
		rpio.Close()
		rpioOpened = false
	}
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

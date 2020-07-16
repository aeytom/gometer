package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/aeytom/gometer/magnet"

	ferraris "github.com/aeytom/gometer/meter"
	"github.com/coreos/go-systemd/daemon"
	client "github.com/influxdata/influxdb1-client"
)

const (
	influxWriteInterval = 30 * time.Second
)

const (
	dfltInfluxURL         = "http://192.168.1.56:8086"
	dfltInfluxUser        = "homemeter"
	dfltInfluxPass        = "istgeheim"
	dfltInfluxDb          = "homemeter"
	dfltInfluxMeasurement = "meter"
)

var (
	// Verbose provide more debugging output
	Verbose *bool
	// Testing do not write influxdb
	Testing *bool
)

var (
	currentMeter ferraris.Ferraris
	solarMeter   ferraris.Ferraris
	gasMeter     magnet.Magnet
)

var (
	influxCon         *client.Client
	influxDatabase    *string
	influxMeasurement *string
)

func main() {
	Verbose = flag.Bool("verbose", false, "provide more debugging output")
	Testing = flag.Bool("test", false, "do not write influxdb")

	argInfluxURL := getEnvArg("INFLUX_URL", "influxUrl", dfltInfluxURL, "influx server url")
	argInfluxUsr := getEnvArg("INFLUX_USER", "influxUser", dfltInfluxUser, "influx db user")
	argInfluxPass := getEnvArg("INFLUX_PASSWORD", "influxPassword", dfltInfluxUser, "influx db user password")
	influxDatabase = getEnvArg("INFLUX_DB", "influxDb", dfltInfluxDb, "influx db name")
	influxMeasurement = getEnvArg("INFLUX_MEASUREMENT", "influxMeasurement", dfltInfluxMeasurement, "influx db measurement")

	argSolar := flag.Float64("solar", 0, "solar meter value")
	argCurrent := flag.Float64("current", 0, "current meter value")
	argGas := flag.Float64("gas", 0, "gas meter value")

	flag.Parse()

	if *Verbose {
		log.Printf("InfluxURL %v, InfluxUsr %v, InfluxPass %v, influxDatabase %v, influxMeasurement %v\n",
			*argInfluxURL, *argInfluxUsr, *argInfluxPass, *influxDatabase, *influxMeasurement)
	}
	magnet.Verbose = *Verbose

	influxURL, err := url.Parse(*argInfluxURL)
	if err != nil {
		log.Fatal(err)
	}
	conf := client.Config{
		URL:       *influxURL,
		Username:  *argInfluxUsr,
		Password:  *argInfluxPass,
		Timeout:   5 * time.Second,
		UserAgent: "gometer/0.3.0",
	}
	influxCon, err = client.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}

	solarMeter = ferraris.New("Solar", 17, 375)
	solarMeter.ResetMeter(float32(*argSolar))
	solarMeter.Print()

	currentMeter = ferraris.New("Current", 27, 75)
	currentMeter.ResetMeter(float32(*argCurrent))
	currentMeter.Print()

	gasMeter = magnet.New("Gas")
	gasMeter.ResetMeter(float32(*argGas))
	gasMeter.Print()

	defer func() {
		solarMeter.Close()
		currentMeter.Close()
		gasMeter.Close()
	}()

	daemon.SdNotify(false, daemon.SdNotifyReady)
	watchdogInterval, _ := daemon.SdWatchdogEnabled(true)
	nextNotify := time.Now().Add(watchdogInterval / 2)

	nextCurrent := time.Now().Add(influxWriteInterval)
	nextSolar := time.Now().Add(influxWriteInterval)
	nextGas := time.Now().Add(influxWriteInterval)

	for {
		if currentMeter.EdgeDetected() {
			currentMeter.Print()
			writeInflux(currentMeter)
			nextCurrent = time.Now().Add(influxWriteInterval)
		} else if time.Now().After(nextCurrent) {
			writeInflux(currentMeter)
			nextCurrent = time.Now().Add(influxWriteInterval)
		}

		if solarMeter.EdgeDetected() {
			solarMeter.Print()
			writeInflux(solarMeter)
			nextSolar = time.Now().Add(influxWriteInterval)
		} else if time.Now().After(nextSolar) {
			writeInflux(solarMeter)
			nextSolar = time.Now().Add(influxWriteInterval)
		}

		if gasMeter.EdgeDetected() {
			gasMeter.Print()
			writeInflux(gasMeter)
			nextGas = time.Now().Add(influxWriteInterval)
		} else if time.Now().After(nextGas) {
			writeInflux(gasMeter)
			nextGas = time.Now().Add(influxWriteInterval)
		}

		if watchdogInterval != 0 && time.Now().After(nextNotify) {
			daemon.SdNotify(false, daemon.SdNotifyWatchdog)
			nextNotify = time.Now().Add(watchdogInterval / 2)
		}

		time.Sleep(time.Millisecond * 50)
	}
}

//
func getEnvArg(env string, arg string, dflt string, usage string) *string {
	ev, avail := os.LookupEnv(env)
	if avail {
		dflt = ev
	}
	v := flag.String(env, dflt, usage)
	return v
}

//
func writeInflux(data interface{}) {
	point := client.Point{
		Time:      time.Now().UTC(),
		Precision: "n",
	}

	if reflect.TypeOf(data) == reflect.TypeOf(ferraris.Ferraris{}) {
		f := data.(ferraris.Ferraris)
		point.Measurement = f.InfluxMeasurement()
		point.Tags = f.InfluxTags()
		point.Fields = f.InfluxFields()
	} else {
		m := data.(magnet.Magnet)
		point.Measurement = m.InfluxMeasurement()
		point.Tags = m.InfluxTags()
		point.Fields = m.InfluxFields()
	}

	if *Testing {
		log.Println(point)
	} else {
		if *Verbose {
			log.Printf("writeInflux %v %v %v\n", point.Tags["meter"], point.Time, point.Fields["value"])
		}
		pts := []client.Point{point}

		bps := client.BatchPoints{
			Points:    pts,
			Database:  *influxDatabase,
			Time:      time.Now().UTC(),
			Precision: "n",
		}
		if *Verbose {
			log.Println(bps)
		}
		go func(bps client.BatchPoints) {
			resp, err := influxCon.Write(bps)
			if err != nil {
				log.Print(err)
			}
			if *Verbose {
				println(resp)
			}
		}(bps)
	}
}

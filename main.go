package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/aeytom/gometer/ferraris"
	"github.com/aeytom/gometer/http"
	"github.com/aeytom/gometer/magnet"
	"github.com/aeytom/gometer/meter"
	"github.com/aeytom/gometer/parameters"
	"github.com/coreos/go-systemd/daemon"
	client "github.com/influxdata/influxdb1-client"
)

const (
	pollInterval        = 50 * time.Millisecond
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
	argInfluxURL := getEnvArg("INFLUX_URL", "influxUrl", dfltInfluxURL, "influx server url")
	argInfluxUsr := getEnvArg("INFLUX_USER", "influxUser", dfltInfluxUser, "influx db user")
	argInfluxPass := getEnvArg("INFLUX_PASSWORD", "influxPassword", dfltInfluxUser, "influx db user password")
	influxDatabase = getEnvArg("INFLUX_DB", "influxDb", dfltInfluxDb, "influx db name")
	influxMeasurement = getEnvArg("INFLUX_MEASUREMENT", "influxMeasurement", dfltInfluxMeasurement, "influx db measurement")

	argSolar := flag.Float64("solar", 0, "solar meter value")
	argCurrent := flag.Float64("current", 0, "current meter value")
	argGas := flag.Float64("gas", 0, "gas meter value")

	http.Flags()
	flag.Parse()

	if *parameters.Verbose {
		log.Printf("InfluxURL %v, InfluxUsr %v, InfluxPass %v, influxDatabase %v, influxMeasurement %v\n",
			*argInfluxURL, *argInfluxUsr, *argInfluxPass, *influxDatabase, *influxMeasurement)
	}
	magnet.Verbose = *parameters.Verbose

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

	solarMeter = ferraris.NewFerraris("Solar", 17, 375, "solar meter value")
	solarMeter.ResetMeter(float32(*argSolar))
	solarMeter.Print()
	http.Add(&solarMeter)
	defer solarMeter.Close()

	currentMeter = ferraris.NewFerraris("Current", 27, 75, "current meter value")
	currentMeter.ResetMeter(float32(*argCurrent))
	currentMeter.Print()
	http.Add(&currentMeter)
	defer currentMeter.Close()

	gasMeter = magnet.NewMagnet("Gas", "gas meter value")
	gasMeter.ResetMeter(float32(*argGas))
	gasMeter.Print()
	http.Add(&gasMeter)
	defer gasMeter.Close()

	daemon.SdNotify(false, daemon.SdNotifyReady)
	watchdogInterval, _ := daemon.SdWatchdogEnabled(true)
	nextNotify := time.Now().Add(watchdogInterval / 2)

	nextCurrent := time.Now().Add(influxWriteInterval)
	nextSolar := time.Now().Add(influxWriteInterval)
	nextGas := time.Now().Add(influxWriteInterval)

	go http.RunServer()

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

		time.Sleep(pollInterval)
	}
}

//
func getEnvArg(env string, arg string, dflt string, usage string) *string {
	ev, avail := os.LookupEnv(env)
	if avail {
		dflt = ev
	}
	v := flag.String(arg, dflt, usage)
	// log.Printf("%s/%s := %s\n", env, arg, *v)
	return v
}

//
func writeInflux(data meter.Meter) {
	point := client.Point{
		Measurement: data.InfluxMeasurement(),
		Tags:        data.InfluxTags(),
		Fields:      data.InfluxFields(),
		Time:        time.Now().UTC(),
		Precision:   "n",
	}

	if *parameters.Testing {
		log.Println(point)
	} else {
		if *parameters.Verbose {
			log.Printf("writeInflux %v %v %v\n", point.Tags["meter"], point.Time, point.Fields["value"])
		}
		pts := []client.Point{point}

		bps := client.BatchPoints{
			Points:    pts,
			Database:  *influxDatabase,
			Time:      time.Now().UTC(),
			Precision: "n",
		}
		if *parameters.Verbose {
			log.Println(bps)
		}
		go func(bps client.BatchPoints) {
			resp, err := influxCon.Write(bps)
			if err != nil {
				log.Print(err)
			}
			if *parameters.Verbose {
				println(resp)
			}
		}(bps)
	}
}

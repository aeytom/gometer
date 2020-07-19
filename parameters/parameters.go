package parameters

import "flag"

var (
	// Verbose provide more debugging output
	Verbose *bool
	// Testing do not write influxdb
	Testing *bool
)

func init() {
	Verbose = flag.Bool("verbose", false, "provide more debugging output")
	Testing = flag.Bool("test", false, "do not write influxdb")
}

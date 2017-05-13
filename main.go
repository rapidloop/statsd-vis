package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const Version = "0.1"

type configType struct {
	webUI       string
	statsdUDP   string
	statsdTCP   string
	flush       time.Duration
	percentiles []int
	retention   time.Duration
}

// config contains the configurable parameters, initialized with default values.
var config = configType{
	webUI:       "0.0.0.0:8080",
	statsdUDP:   "127.0.0.1:8125",
	statsdTCP:   "127.0.0.1:8125",
	flush:       10 * time.Second,
	percentiles: []int{90, 95, 99},
	retention:   30 * time.Minute,
}

var (
	data        *StatsRing
	names       = NewMetricNames()
	webUI       = flag.String("webui", config.webUI, "web UI listen `address`")
	statsdUDP   = flag.String("statsdudp", config.statsdUDP, "statsd UDP listen `address`")
	statsdTCP   = flag.String("statsdtcp", config.statsdTCP, "statsd TCP listen `address`")
	flush       = flag.Duration("flush", config.flush, "flush `interval`")
	percentiles = flag.String("percentiles", "90,95,99", "percentiles for timer metrics")
	retention   = flag.Duration("retention", config.retention, "`duration` to retain the metrics for")
)

func usage() {
	fmt.Fprintf(os.Stderr,
		`statsd-vis %s - (c) 2017 RapidLoop - MIT Licensed - https://statsd-vis.info/
statd-vis is a standalone statsd server with built-in visualization

`, Version)
	flag.PrintDefaults()
}

func intarray(s string) (r []int) {
	parts := strings.Split(s, ",")
	r = make([]int, len(parts))
	for i, p := range parts {
		if v, err := strconv.Atoi(strings.TrimSpace(p)); err != nil {
			log.Fatalf("invalid percentiles string: %v", err)
		} else if v <= 0 || v >= 100 {
			log.Fatalf("invalid percentile %d, must be > 0 and < 100", v)
		} else {
			r[i] = v
		}
	}
	if len(parts) == 0 {
		log.Fatal("invalid percentiles string")
	}
	sort.Ints(r)
	return
}

func main() {

	// parse command line
	flag.Usage = usage
	flag.Parse()
	config.webUI = *webUI
	config.statsdUDP = *statsdUDP
	config.statsdTCP = *statsdTCP
	config.flush = *flush
	config.percentiles = intarray(*percentiles)
	config.retention = *retention

	// set log flags
	log.SetPrefix("statsd-vis: ")
	log.SetFlags(0)

	// start the statsd server
	data = NewStatsRing(int(config.retention / config.flush))
	startStatsd()
	log.Printf("statsd UDP server started, listening on %s", config.statsdUDP)
	log.Printf("statsd TCP server started, listening on %s", config.statsdTCP)
	log.Printf("config: flush interval=%v, retention=%v, percentiles=%v",
		config.flush, config.retention, config.percentiles)

	// start the web server
	go startWeb()
	log.Printf("web server started, listening on %s", config.webUI)

	// wait for ^C
	log.Printf("Hit ^C to exit..")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch)
	for s := range ch {
		if s == syscall.SIGTERM || s == os.Interrupt {
			break
		}
	}
	signal.Stop(ch)
	close(ch)
	log.Print("Bye.")
}

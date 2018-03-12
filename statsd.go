package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

const queueLen = 1000

type HoldingArea struct {
	counters map[string]int64
	timers   map[string]timerInfo
	gauges   map[string]int64
	sets     map[string]map[string]bool
}

func (h *HoldingArea) clear() {
	h.counters = make(map[string]int64)
	h.timers = make(map[string]timerInfo)
	h.sets = make(map[string]map[string]bool)
	h.gauges = make(map[string]int64)
}

type timerInfo struct {
	values []float64
	count  int64
}

var (
	udpConn *net.UDPConn
	tcpLis  *net.TCPListener
	queue   chan sdop
	area    HoldingArea
)

func startStatsd() {
	udpAddr, err := net.ResolveUDPAddr("udp", config.statsdUDP)
	if err != nil {
		log.Fatalf("statsd udp listen address: %v", err)
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", config.statsdTCP)
	if err != nil {
		log.Fatalf("statsd tcp listen address: %v", err)
	}

	udpConn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("statsd udp listen: %v", err)
	}

	tcpLis, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatalf("statsd tcp listen: %v", err)
	}

	queue = make(chan sdop, queueLen)
	go aggregator()
	go udpHandler()
	go tcpHandler()
}

func udpHandler() {
	buf := make([]byte, 16384)
	for {
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			if addr != nil {
				log.Printf("statsd udp read error from %v: %v", addr, err)
			} else {
				log.Printf("statsd udp read error: %v", err)
			}
			break
		} else if n == 0 {
			log.Printf("statsd udp read 0 bytes from %v", addr)
		} else {
			if bytes.IndexByte(buf[:n], '\n') == -1 {
				// optimization: typical single-line packets
				parseLineToQueue(string(buf[:n]), addr)
			} else {
				// costly, generic, multi-line scanner
				parseToQueue(bytes.NewBuffer(buf[:n]), addr)
			}
			buf = buf[:len(buf)]
		}
	}
	udpConn.Close()
}

func tcpHandler() {
	for {
		tcpConn, err := tcpLis.AcceptTCP()
		if err != nil {
			log.Printf("statsd tcp accept error: %v", err)
			break
		} else {
			go tcpClientHandler(tcpConn)
		}
	}
}

func tcpClientHandler(tcpConn *net.TCPConn) {
	rip := tcpConn.RemoteAddr()
	parseToQueue(tcpConn, rip)
	tcpConn.Close()
}

func parseToQueue(r io.Reader, rip net.Addr) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parseLineToQueue(scanner.Text(), rip)
	}
}

// gorets:1|c
// gorets:1|c|@0.1
// glork:320|ms
// glork:320|ms|@0.1
// gaugor:333|g
// gaugor:-10|g
// uniques:765|s

func parseLineToQueue(line string, rip net.Addr) {
	var err error
	colon := strings.Index(line, ":")
	bar1 := strings.Index(line, "|")
	if colon < 1 || bar1 < colon || bar1 == colon+1 || bar1 == len(line)-1 {
		log.Printf("bad line [%s] from ip [%v]", line, rip)
		return
	}
	typeEnd := len(line)
	bar2 := strings.Index(line[bar1+1:], "|")
	sampleRate := math.NaN()
	if bar2 != -1 {
		typeEnd = bar1 + 1 + bar2
		rest := line[bar1+1+bar2+1:]
		// sampling format
		if rest[0] == '@' {
			sampleEnd := strings.Index(rest[0:], "|")
			if sampleRate, err = strconv.ParseFloat(rest[1:sampleEnd], 64); err != nil {
				log.Printf("bad line [%s] from ip [%v], err: %s", line, rip, err)
				return
			}
		} else if rest[0] == '#' {
			// tag format
			// TODO: do something with the tags, now simply tolerate them
			// tags := rest[1:]
			// log.Printf("tags: %s", tags)
		} else {
		 	log.Printf("bad line [%s] from ip [%v]", line, rip)
		 	return
		}
	}

	// op is the operation that we're parsing into
	op := sdop{name: line[0:colon], rate: sampleRate}

	value := line[colon+1 : bar1]
	switch line[bar1+1 : typeEnd] {
	case "c":
		ival, err := strconv.ParseInt(value, 10, 64)
		if err != nil || ival < 0 {
			log.Printf("bad line [%s] from ip [%v]", line, rip)
			return
		}
		op.op = SDOP_C_ADD
		op.ival = ival
		//log.Printf("counter: %s=%d @ %.2f", line[0:colon], ival, sampleRate)
	case "ms":
		fval, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("bad line [%s] from ip [%v]", line, rip)
			return
		}
		op.op = SDOP_T
		op.fval = fval
		//log.Printf("timer: %s=%.2f @ %.2f", line[0:colon], fval, sampleRate)
	case "g":
		if strings.HasPrefix(value, "+") {
			op.op = SDOP_G_INCR
		} else if strings.HasPrefix(value, "-") {
			op.op = SDOP_G_DECR
		} else {
			op.op = SDOP_G_SET
		}
		ival, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Printf("bad line [%s] from ip [%v]", line, rip)
			return
		}
		op.ival = ival
		// log.Printf("gauge: op=%d %s=%d", gop, line[0:colon], ival)
	case "s":
		//log.Printf("set: %s=%s", line[0:colon], value)
		op.op = SDOP_S
		op.sval = value
	case "h":
		// log.Printf("histogram: %s=%s", line[0:colon], value)
		ival, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Printf("bad line [%s] from ip [%v]", line, rip)
			return
		}
		// TODO: properly support histogram counters
		// http://docs.datadoghq.com/guides/metrics/#histograms
		op.op = SDOP_G_SET
		op.ival = ival
	default:
		log.Printf("bad line [%s] from ip [%v], unknown %s", line, rip, line[bar1 + 1 : typeEnd])
		return
	}

	queue <- op
}

// operations:
// 1. add to counter [name] intvalue [ival] sample rate [srate]
// 2. add to timer set of [name] floatvalue [fval] sample rate [srate]
// 3. set gauge [name] to intvalue [ival]
// 4. add to gauge [name] intvalue [ival]
// 5. reduce from gauge [name] intvalue [ival]
// 6. add to set [name] value strvalue [sval]

const (
	SDOP_C_ADD = iota
	SDOP_T
	SDOP_G_SET
	SDOP_G_INCR
	SDOP_G_DECR
	SDOP_S
)

type sdop struct {
	op   int
	name string
	ival int64
	fval float64
	sval string
	rate float64
}

// The aggregator thread.
func aggregator() {
	// setup
	area.clear()
	timer := time.NewTicker(config.flush)

	for {
		select {
		case op := <-queue:
			switch op.op {
			case SDOP_C_ADD:
				count := op.ival
				if !math.IsNaN(op.rate) && op.rate != 0 {
					count = int64(float64(op.ival) / op.rate)
				}
				if v, ok := area.counters[op.name]; ok {
					area.counters[op.name] = v + count
				} else {
					area.counters[op.name] = count
				}
			case SDOP_T:
				count := int64(1)
				if !math.IsNaN(op.rate) && op.rate != 0 {
					count = int64(1.0 / op.rate)
				}
				if v, ok := area.timers[op.name]; ok {
					v.values = append(v.values, op.fval)
					v.count += count
					area.timers[op.name] = v
				} else {
					v.values = []float64{op.fval}
					v.count = count
					area.timers[op.name] = v
				}
			case SDOP_G_SET:
				area.gauges[op.name] = op.ival
			case SDOP_G_INCR:
				if v, ok := area.gauges[op.name]; ok {
					area.gauges[op.name] = v + op.ival
				} else {
					area.gauges[op.name] = op.ival
				}
			case SDOP_G_DECR:
				if v, ok := area.gauges[op.name]; ok {
					area.gauges[op.name] = v - op.ival
				} else {
					// CFG: statsdaemon floors value at 0, statsd does not(?)
					area.gauges[op.name] = -op.ival
				}
			case SDOP_S:
				if v, ok := area.sets[op.name]; ok {
					v[op.sval] = true
				} else {
					area.sets[op.name] = map[string]bool{op.sval: true}
				}
			}
		case <-timer.C:
			statsdFlush()
		}
	}
}

// get the p'th percentile value from the sorted list v
func percentile(v []float64, p int) float64 {
	s := (float64(p) / 100.0) * float64(len(v))
	r := int(math.Floor(s + 0.5))
	if r > 0 && r <= len(v) {
		return v[r-1]
	} else {
		return math.NaN()
	}
}

func statsdFlush() {
	result := Stats{
		At:      time.Now(),
		Metrics: make(map[string]float64),
	}
	//log.Printf("flush @ %v", result.At)
	for bucket, value := range area.counters {
		//log.Printf("counter: %s = %.2f", bucket, float64(value))
		result.add(bucket, float64(value))
	}
	for bucket, tinfo := range area.timers {
		values := tinfo.values
		total := 0.0
		min := math.Inf(+1)
		max := math.Inf(-1)
		for _, v := range values {
			total += v
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		mean := total / float64(len(values))
		// sort the values
		sort.Float64s(values)
		if len(values) > 1 {
			for _, pile := range config.percentiles {
				if pilev := percentile(values, pile); !math.IsNaN(pilev) {
					metric := fmt.Sprintf("%s.upper_%d", bucket, pile)
					//log.Printf("timer: %s = %.2f", metric, pilev)
					result.add(metric, pilev)
					names.AddTimerGen(metric)
				}
			}
		}
		//log.Printf("timer: %s = %.2f", bucket+".mean", mean)
		result.add(bucket+".mean", mean)
		names.AddTimerGen(bucket + ".mean")
		//log.Printf("timer: %s = %.2f", bucket+".lower", min)
		result.add(bucket+".lower", min)
		names.AddTimerGen(bucket + ".lower")
		//log.Printf("timer: %s = %.2f", bucket+".upper", max)
		result.add(bucket+".upper", max)
		names.AddTimerGen(bucket + ".upper")
		//log.Printf("timer: %s = %.2f", bucket+".count", float64(tinfo.count))
		result.add(bucket+".count", float64(tinfo.count))
		names.AddTimerGen(bucket + ".count")
	}
	for bucket, value := range area.gauges {
		//log.Printf("gauge: %s = %.2f", bucket, float64(value))
		result.add(bucket, float64(value))
	}
	for bucket, value := range area.sets {
		//log.Printf("set: %s = %.2f", bucket, float64(len(value)))
		result.add(bucket, float64(len(value)))
	}
	// store the result
	names.Add(&area)
	data.Add(&result)
	// empty the buckets
	area.clear()
}

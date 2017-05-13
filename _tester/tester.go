package main

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	go statsdSender()

	ticker := time.NewTicker(time.Second)
	i := 0
	for {
		select {
		case <-ticker.C:
			r := rand.Intn(10000)
			// some counters
			StatCount("test.counter.1.rand", uint(r))
			StatCount("test.counter.2.1persec", 1)
			StatCount("test.counter.3.5persec", 5)
			if i%60 == 0 {
				StatCount("test.counter.4.1permin", 1)
			}
			StatCountRate("test.counter.5.1persec.rate0.1", 1, 0.1)
			// some gauges
			StatGauge("test.gauge.1.rand", r)
			StatGauge("test.gauge.2.fixedat5", 5)
			StatGauge("test.gauge.3.fixedat5.2", 10)
			StatGauge("test.gauge.4.fixedat5.2", 5)
			if i%60 == 0 {
				StatGauge("test.guage.5.rand.1permin", r)
			}
			// some timers
			StatTime("test.timer.1.rand", time.Millisecond*time.Duration(r))
			StatTime("test.timer.2.fixed500", time.Millisecond*500)
			StatTimeRate("test.timer.3.fixed500.rate0.1", time.Millisecond*500, 0.1)
			for i := 1; i <= 100; i++ {
				StatTime("test.timer.4.upto100", time.Millisecond*time.Duration(i))
			}
			// sets
			for i := 0; i < rand.Intn(30); i++ {
				StatSet("test.set.1.rand30", strconv.Itoa(i))
			}
			for i := 0; i < 10; i++ {
				StatSet("test.set.2.fixed10", strconv.Itoa(i))
			}
			// done
			i++
		}
	}
}

var queue = make(chan string, 100)

func StatCount(metric string, value uint) {
	queue <- fmt.Sprintf("%s:%d|c", metric, value)
}

func StatCountRate(metric string, value uint, rate float64) {
	queue <- fmt.Sprintf("%s:%d|c|@%.2f", metric, value, rate)
}

func StatTime(metric string, took time.Duration) {
	ms := float64(took) / 1e6
	queue <- fmt.Sprintf("%s:%.2f|ms", metric, ms)
}

func StatTimeRate(metric string, took time.Duration, rate float64) {
	ms := float64(took) / 1e6
	queue <- fmt.Sprintf("%s:%.2f|ms|@%.2f", metric, ms, rate)
}

func StatGauge(metric string, value int) {
	queue <- fmt.Sprintf("%s:%d|g", metric, value)
}

func StatSet(metric, value string) {
	queue <- fmt.Sprintf("%s:%s|s", metric, value)
}

func statsdSender() {
	for s := range queue {
		if conn, err := net.Dial("udp", "127.0.0.1:8125"); err == nil {
			io.WriteString(conn, s)
			conn.Close()
		}
	}
}

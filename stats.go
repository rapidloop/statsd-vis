package main

import (
	"html/template"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Stats struct {
	At      time.Time
	Metrics map[string]float64
}

func (s *Stats) add(k string, v float64) {
	s.Metrics[k] = v
}

// StatsRing is a ring buffer of Stats objects. Kind of.
type StatsRing struct {
	sync.Mutex
	Values []*Stats
	Head   int
}

func NewStatsRing(n int) *StatsRing {
	return &StatsRing{Values: make([]*Stats, n)}
}

func (r *StatsRing) Add(s *Stats) {
	r.Lock()
	defer r.Unlock()
	r.Values[r.Head] = s
	r.Head = (r.Head + 1) % len(r.Values)
}

type GraphData struct {
	Idx        int
	Title      string
	Metrics    []string
	Datapoints []Datapoint
}

type Datapoint struct {
	At     time.Time
	Values []float64
}

func (d *Datapoint) ValuesStr() template.JS {
	parts := make([]string, len(d.Values))
	for i, v := range d.Values {
		if math.IsNaN(v) {
			parts[i] = "null"
		} else {
			parts[i] = strconv.FormatFloat(v, 'f', -1, 64)
		}
	}
	return template.JS(strings.Join(parts, ", "))
}

func (r *StatsRing) GetDataForGraph(names []string) (g GraphData) {
	r.Lock()
	defer r.Unlock()
	g.Metrics = names
	g.Datapoints = make([]Datapoint, 0, len(r.Values))
	pos := r.Head
	for {
		pos = (pos + 1) % len(r.Values)
		if pos == r.Head {
			return
		}
		if r.Values[pos] != nil {
			m := r.Values[pos].Metrics
			dp := Datapoint{
				At:     r.Values[pos].At,
				Values: make([]float64, len(names)),
			}
			allNaN := true
			for i, n := range names {
				if v, found := m[n]; found {
					dp.Values[i] = v
					allNaN = false
				} else {
					dp.Values[i] = math.NaN()
				}
			}
			if !allNaN {
				g.Datapoints = append(g.Datapoints, dp)
			}
		}
	}
}

const (
	mtCounter = iota
	mtTimer
	mtGauge
	mtSet
	mtTimerGen
)

type MetricNames struct {
	Names map[string]int
	sync.Mutex
}

func NewMetricNames() *MetricNames {
	return &MetricNames{Names: make(map[string]int)}
}

func (m *MetricNames) List() (out [][]string) {
	m.Lock()
	out = make([][]string, 4)
	for n, t := range m.Names {
		if t != mtTimerGen {
			out[t] = append(out[t], n)
		}
	}
	m.Unlock()
	return
}

func (m *MetricNames) Add(a *HoldingArea) {
	m.Lock()
	for n, _ := range a.counters {
		m.Names[n] = mtCounter
	}
	for n, _ := range a.timers {
		m.Names[n] = mtTimer
	}
	for n, _ := range a.gauges {
		m.Names[n] = mtGauge
	}
	for n, _ := range a.sets {
		m.Names[n] = mtSet
	}
	m.Unlock()
}

func (m *MetricNames) AddTimerGen(n string) {
	m.Lock()
	m.Names[n] = mtTimerGen
	m.Unlock()
}

func (m *MetricNames) Find(r string) (out []string) {
	m.Lock()
	for n, _ := range m.Names {
		if strings.HasPrefix(n, r) {
			out = append(out, n)
		}
	}
	m.Unlock()
	sort.Strings(out)
	return
}

func (m *MetricNames) FindAll(rs []string) (out []string) {
	for _, r := range rs {
		out = append(out, m.Find(r)...)
	}
	return
}

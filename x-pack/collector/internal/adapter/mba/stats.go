package mba

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/monitoring"
)

// stats bundles common metricset stats.
type stats struct {
	key      string          // full stats key
	ref      uint32          // number of modules/metricsets reusing stats instance
	success  *monitoring.Int // Total success events.
	failures *monitoring.Int // Total error events.
	events   *monitoring.Int // Total events published.
}

var (
	fetchesLock = sync.Mutex{}
	fetches     = map[string]*stats{}
)

// Expvar metric names.
const (
	successesKey = "success"
	failuresKey  = "failures"
	eventsKey    = "events"
)

func getMetricSetStats(key string) *stats {
	fetchesLock.Lock()
	defer fetchesLock.Unlock()

	if s := fetches[key]; s != nil {
		s.ref++
		return s
	}

	reg := monitoring.Default.NewRegistry(key)
	s := &stats{
		key:      key,
		ref:      1,
		success:  monitoring.NewInt(reg, successesKey),
		failures: monitoring.NewInt(reg, failuresKey),
		events:   monitoring.NewInt(reg, eventsKey),
	}

	fetches[key] = s
	return s
}

func releaseStats(s *stats) {
	fetchesLock.Lock()
	defer fetchesLock.Unlock()

	s.ref--
	if s.ref > 0 {
		return
	}

	delete(fetches, s.key)
	monitoring.Default.Remove(s.key)
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// concurrencyObserver tracks the effective concurrency level based on pipeline
// backpressure using additive-increase/multiplicative-decrease (AIMD):
//
//   - When a worker completes without experiencing backpressure (publish did
//     not block for longer than publishLatencyThreshold), the observer
//     considers the pipeline healthy and may increase the recorded level.
//   - When backpressure is detected (publish blocked), the observer reduces
//     the recorded level.
//
// The observer does not gate admission. The semaphore in the run loop is
// the hard ceiling. The AIMD level records what the system converges to so
// operators can see the effective concurrency and tune number_of_workers.
type concurrencyObserver struct {
	maxWorkers int
	log        *logp.Logger

	// mu protects adjustment decisions to avoid racing scale events.
	mu             sync.Mutex
	lastAdjust     time.Time
	adjustCooldown time.Duration

	// Monitoring metrics.
	level     *monitoring.Int
	scaleUps  *monitoring.Uint
	scaleDown *monitoring.Uint
}

// concurrencyObserverConfig holds tuning for the observer.
// AdjustCooldown prevents oscillation; 5s is a sensible production default.
type concurrencyObserverConfig struct {
	MaxWorkers     int
	AdjustCooldown time.Duration
	Log            *logp.Logger
	Registry       *monitoring.Registry
}

func newConcurrencyObserver(cfg concurrencyObserverConfig) *concurrencyObserver {
	if cfg.MaxWorkers < 1 {
		cfg.MaxWorkers = 1
	}
	if cfg.AdjustCooldown < 0 {
		cfg.AdjustCooldown = 0
	}
	if cfg.Log == nil {
		cfg.Log = logp.NewNopLogger()
	}
	if cfg.Registry == nil {
		cfg.Registry = monitoring.NewRegistry()
	}

	initial := max(cfg.MaxWorkers/2, 1)

	o := &concurrencyObserver{
		maxWorkers:     cfg.MaxWorkers,
		adjustCooldown: cfg.AdjustCooldown,
		log:            cfg.Log,
		level:          monitoring.NewInt(cfg.Registry, "concurrency_level"),
		scaleUps:       monitoring.NewUint(cfg.Registry, "concurrency_scale_ups_total"),
		scaleDown:      monitoring.NewUint(cfg.Registry, "concurrency_scale_downs_total"),
	}
	o.level.Set(int64(initial))
	return o
}

// OnSuccess signals that a unit of work completed without backpressure.
// The observer may increase the recorded concurrency level.
func (o *concurrencyObserver) OnSuccess() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if time.Since(o.lastAdjust) < o.adjustCooldown {
		return
	}
	cur := int(o.level.Get())
	if cur < o.maxWorkers {
		next := cur + 1
		o.level.Set(int64(next))
		o.lastAdjust = time.Now()
		o.scaleUps.Inc()
		o.log.Infow("Concurrency increased.", "from", cur, "to", next, "max", o.maxWorkers)
	}
}

// OnBackpressure signals that publishing blocked, indicating the pipeline
// is saturated. The observer reduces the recorded level (multiplicative decrease).
func (o *concurrencyObserver) OnBackpressure() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if time.Since(o.lastAdjust) < o.adjustCooldown {
		return
	}
	cur := int(o.level.Get())
	next := max(cur/2, 1)
	if next != cur {
		o.level.Set(int64(next))
		o.lastAdjust = time.Now()
		o.scaleDown.Inc()
		o.log.Infow("Concurrency decreased (backpressure).", "from", cur, "to", next, "max", o.maxWorkers)
	}
}

// ScaleUps returns the total number of scale-up events.
func (o *concurrencyObserver) ScaleUps() uint64 { return o.scaleUps.Get() }

// ScaleDowns returns the total number of scale-down events.
func (o *concurrencyObserver) ScaleDowns() uint64 { return o.scaleDown.Get() }

// publishWithBackpressure wraps a publish function and signals the observer
// when backpressure is detected (publish takes longer than the threshold).
func publishWithBackpressure(co *concurrencyObserver, threshold time.Duration, publish func()) {
	start := time.Now()
	publish()
	if time.Since(start) > threshold {
		co.OnBackpressure()
	} else {
		co.OnSuccess()
	}
}

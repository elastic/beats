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

// concurrencyController dynamically adjusts the number of concurrent workers
// between 1 and maxWorkers based on pipeline backpressure. It uses a simple
// additive-increase/multiplicative-decrease (AIMD) approach:
//
//   - When a worker completes without experiencing backpressure (publish did
//     not block for longer than publishLatencyThreshold), the controller
//     considers the pipeline healthy and may increase concurrency.
//   - When backpressure is detected (publish blocked), the controller reduces
//     concurrency.
//
// Config values (number_of_workers) become the upper bound rather than a fixed
// pool size. Users who previously tuned this knob get the same ceiling, but the
// controller finds the right operating point automatically.
type concurrencyController struct {
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

// concurrencyControllerConfig holds tuning for the controller.
// AdjustCooldown prevents oscillation; 5s is a sensible production default.
type concurrencyControllerConfig struct {
	MaxWorkers     int
	AdjustCooldown time.Duration
	Log            *logp.Logger
	Registry       *monitoring.Registry
}

func newConcurrencyController(cfg concurrencyControllerConfig) *concurrencyController {
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

	cc := &concurrencyController{
		maxWorkers:     cfg.MaxWorkers,
		adjustCooldown: cfg.AdjustCooldown,
		log:            cfg.Log,
		level:          monitoring.NewInt(cfg.Registry, "concurrency_level"),
		scaleUps:       monitoring.NewUint(cfg.Registry, "concurrency_scale_ups_total"),
		scaleDown:      monitoring.NewUint(cfg.Registry, "concurrency_scale_downs_total"),
	}
	cc.level.Set(int64(initial))
	return cc
}

// Current returns the current allowed concurrency level.
func (cc *concurrencyController) Current() int {
	return int(cc.level.Get())
}

// Acquire blocks until a concurrency slot is available or returns false if
// the provided done channel is closed. This is a simple semaphore-style gate.
func (cc *concurrencyController) Acquire(done <-chan struct{}, sem chan struct{}) bool {
	select {
	case sem <- struct{}{}:
		return true
	case <-done:
		return false
	}
}

// Release returns a concurrency slot.
func (cc *concurrencyController) Release(sem chan struct{}) {
	<-sem
}

// OnSuccess signals that a unit of work completed without backpressure.
// The controller may increase concurrency.
func (cc *concurrencyController) OnSuccess() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if time.Since(cc.lastAdjust) < cc.adjustCooldown {
		return
	}
	cur := int(cc.level.Get())
	if cur < cc.maxWorkers {
		next := cur + 1
		cc.level.Set(int64(next))
		cc.lastAdjust = time.Now()
		cc.scaleUps.Inc()
		cc.log.Infow("Concurrency increased.", "from", cur, "to", next, "max", cc.maxWorkers)
	}
}

// OnBackpressure signals that publishing blocked, indicating the pipeline
// is saturated. The controller reduces concurrency (multiplicative decrease).
func (cc *concurrencyController) OnBackpressure() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if time.Since(cc.lastAdjust) < cc.adjustCooldown {
		return
	}
	cur := int(cc.level.Get())
	next := max(cur/2, 1)
	if next != cur {
		cc.level.Set(int64(next))
		cc.lastAdjust = time.Now()
		cc.scaleDown.Inc()
		cc.log.Infow("Concurrency decreased (backpressure).", "from", cur, "to", next, "max", cc.maxWorkers)
	}
}

// ScaleUps returns the total number of scale-up events.
func (cc *concurrencyController) ScaleUps() uint64 { return cc.scaleUps.Get() }

// ScaleDowns returns the total number of scale-down events.
func (cc *concurrencyController) ScaleDowns() uint64 { return cc.scaleDown.Get() }

// publishWithBackpressure wraps a publish function and signals the controller
// when backpressure is detected (publish takes longer than the threshold).
func publishWithBackpressure(cc *concurrencyController, threshold time.Duration, publish func()) {
	start := time.Now()
	publish()
	if time.Since(start) > threshold {
		cc.OnBackpressure()
	} else {
		cc.OnSuccess()
	}
}

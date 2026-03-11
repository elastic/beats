// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"
)

var (
	systemMu   sync.Mutex
	systemInst *SystemRegistryBridge
	systemRefs int
)

// SystemRegistryBridge bridges process-level system metrics (beat.memstats.*,
// beat.cpu.*, system.load.*, etc.) into OTel once per process. These metrics
// are identical regardless of which receiver reports them, so a singleton
// avoids duplication.
type SystemRegistryBridge struct {
	logger        *zap.Logger
	meter         metric.Meter
	statsRegistry *monitoring.Registry

	mu           sync.Mutex
	registration metric.Registration
	closed       bool

	intGauges   map[string]metric.Int64ObservableGauge
	intCounters map[string]metric.Int64ObservableCounter
	floatGauges map[string]metric.Float64ObservableGauge
}

// AcquireSystemBridge returns a release function that must be called during
// shutdown. On first call it creates the singleton SystemRegistryBridge with
// its own monitoring registry populated by SetupMetricsOptions. Subsequent
// calls increment a reference count. When all callers have released, the
// bridge is shut down.
//
// It is safe for the singleton to use the TelemetrySettings from whichever
// receiver calls first, even if that receiver later shuts down. The OTel SDK
// MeterProvider and its async callbacks remain functional for the lifetime of
// the provider — instrument registrations and observations are not scoped to
// the component that created them. The provider is owned by the collector
// service and outlives individual receivers.
func AcquireSystemBridge(settings component.TelemetrySettings) (func(), error) {
	systemMu.Lock()
	defer systemMu.Unlock()

	if systemInst == nil {
		reg := monitoring.NewRegistry()
		processReg := reg.GetOrCreateRegistry("beat")
		systemReg := reg.GetOrCreateRegistry("system")

		err := metricreport.SetupMetricsOptions(metricreport.MetricOptions{
			Logger:         logp.NewLogger("system-bridge"),
			Name:           "beat",
			SystemMetrics:  systemReg,
			ProcessMetrics: processReg,
		})
		if err != nil {
			return nil, err
		}

		bridge, err := newSystemRegistryBridge(settings, reg)
		if err != nil {
			return nil, err
		}
		systemInst = bridge
	}

	systemRefs++

	var once sync.Once
	return func() {
		once.Do(func() {
			systemMu.Lock()
			defer systemMu.Unlock()

			systemRefs--
			if systemRefs <= 0 {
				systemInst.shutdown()
				systemInst = nil
				systemRefs = 0
			}
		})
	}, nil
}

// newSystemRegistryBridge creates a SystemRegistryBridge from the given
// registry. The registry should already be populated with system metrics
// (via SetupMetricsOptions or manually for tests).
func newSystemRegistryBridge(settings component.TelemetrySettings, statsRegistry *monitoring.Registry) (*SystemRegistryBridge, error) {
	mp := settings.MeterProvider
	if mp == nil {
		mp = noop.NewMeterProvider()
	}

	logger := settings.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	b := &SystemRegistryBridge{
		logger:        logger,
		meter:         mp.Meter(scopeName),
		statsRegistry: statsRegistry,
		intGauges:     make(map[string]metric.Int64ObservableGauge),
		intCounters:   make(map[string]metric.Int64ObservableCounter),
		floatGauges:   make(map[string]metric.Float64ObservableGauge),
	}

	if statsRegistry == nil {
		return b, nil
	}

	// Discover all metrics currently in the registry.
	snap := monitoring.CollectFlatSnapshot(statsRegistry, monitoring.Full, false)
	for key := range snap.Ints {
		if isGauge(key) {
			inst, err := b.meter.Int64ObservableGauge(key)
			if err != nil {
				return nil, err
			}
			b.intGauges[key] = inst
		} else {
			inst, err := b.meter.Int64ObservableCounter(key)
			if err != nil {
				return nil, err
			}
			b.intCounters[key] = inst
		}
	}
	for key := range snap.Floats {
		inst, err := b.meter.Float64ObservableGauge(key)
		if err != nil {
			return nil, err
		}
		b.floatGauges[key] = inst
	}

	logger.Info("system bridge discovered metrics",
		zap.Int("int_gauges", len(b.intGauges)),
		zap.Int("int_counters", len(b.intCounters)),
		zap.Int("float_gauges", len(b.floatGauges)),
	)

	// Register the callback with all instruments.
	instruments := b.allInstruments()
	if len(instruments) > 0 {
		reg, err := b.meter.RegisterCallback(b.callback, instruments...)
		if err != nil {
			return nil, err
		}
		b.registration = reg
	}

	return b, nil
}

// allInstruments returns all known instruments as a slice of metric.Observable.
func (b *SystemRegistryBridge) allInstruments() []metric.Observable {
	out := make([]metric.Observable, 0,
		len(b.intGauges)+len(b.intCounters)+len(b.floatGauges))
	for _, inst := range b.intGauges {
		out = append(out, inst)
	}
	for _, inst := range b.intCounters {
		out = append(out, inst)
	}
	for _, inst := range b.floatGauges {
		out = append(out, inst)
	}
	return out
}

// callback takes a flat snapshot and observes all values. Unknown keys
// (registered after construction) are silently skipped.
func (b *SystemRegistryBridge) callback(_ context.Context, obs metric.Observer) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	if b.statsRegistry == nil {
		return nil
	}

	snap := monitoring.CollectFlatSnapshot(b.statsRegistry, monitoring.Full, false)

	for key, value := range snap.Ints {
		if inst, ok := b.intGauges[key]; ok {
			obs.ObserveInt64(inst, value)
			continue
		}
		if inst, ok := b.intCounters[key]; ok {
			obs.ObserveInt64(inst, value)
		}
	}
	for key, value := range snap.Floats {
		if inst, ok := b.floatGauges[key]; ok {
			obs.ObserveFloat64(inst, value)
		}
	}

	return nil
}

// shutdown unregisters the callback and marks the bridge as closed.
func (b *SystemRegistryBridge) shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	if b.registration != nil {
		_ = b.registration.Unregister()
		b.registration = nil
	}
}

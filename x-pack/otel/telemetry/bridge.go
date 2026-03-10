// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"

	logreport "github.com/elastic/beats/v7/libbeat/monitoring/report/log"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const scopeName = "github.com/elastic/beats/v7/x-pack/otel/telemetry"

// RegistryBridge dynamically discovers metrics from beats monitoring registries
// and bridges them into OTel async instruments. Instruments are auto-created
// from FlatSnapshot/StructSnapshot keys — no hardcoded metric mappings.
type RegistryBridge struct {
	logger         *zap.Logger
	meter          metric.Meter
	receiverID     string
	statsRegistry  *monitoring.Registry
	inputsRegistry *monitoring.Registry
	statsAttrs     metric.MeasurementOption

	mu           sync.RWMutex
	registration metric.Registration
	reRegWg      sync.WaitGroup
	reRegMu      sync.Mutex // serializes createAndReRegister goroutines
	closed       bool

	// Stats instruments keyed by registry key path.
	intGauges   map[string]metric.Int64ObservableGauge
	intCounters map[string]metric.Int64ObservableCounter
	floatGauges map[string]metric.Float64ObservableGauge

	// Per-input instruments keyed by metric field name.
	inputIntGauges   map[string]metric.Int64ObservableGauge
	inputIntCounters map[string]metric.Int64ObservableCounter
	inputFloatGauges map[string]metric.Float64ObservableGauge

	// Pending keys discovered during callback that need instrument creation.
	// These are processed by an async goroutine since the OTel SDK pipeline
	// lock is held during callback execution, preventing instrument creation.
	pendingStatsInts   []string
	pendingStatsFloats []string
	pendingInputInts   []string
	pendingInputFloats []string
}

// NewRegistryBridge creates a RegistryBridge that discovers all current metrics
// from the given registries and registers a single OTel async callback.
// The receiverID is used as a "receiver" attribute on all observations so that
// multiple receivers (e.g., filebeat + metricbeat) don't collide.
func NewRegistryBridge(settings component.TelemetrySettings, receiverID string, statsRegistry, inputsRegistry *monitoring.Registry) (*RegistryBridge, error) {
	mp := settings.MeterProvider
	if mp == nil {
		mp = noop.NewMeterProvider()
	}

	logger := settings.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	b := &RegistryBridge{
		logger:           logger,
		meter:            mp.Meter(scopeName),
		receiverID:       receiverID,
		statsRegistry:    statsRegistry,
		inputsRegistry:   inputsRegistry,
		statsAttrs:       metric.WithAttributeSet(attribute.NewSet(attribute.String("receiver", receiverID))),
		intGauges:        make(map[string]metric.Int64ObservableGauge),
		intCounters:      make(map[string]metric.Int64ObservableCounter),
		floatGauges:      make(map[string]metric.Float64ObservableGauge),
		inputIntGauges:   make(map[string]metric.Int64ObservableGauge),
		inputIntCounters: make(map[string]metric.Int64ObservableCounter),
		inputFloatGauges: make(map[string]metric.Float64ObservableGauge),
	}

	// Discover initial stats and per-input metrics under a single write lock.
	// Only ints and floats are bridged — OTel instruments are numeric only.
	b.mu.Lock()
	if statsRegistry != nil {
		snap := monitoring.CollectFlatSnapshot(statsRegistry, monitoring.Full, false)
		for key := range snap.Ints {
			if err := b.ensureStatsInt(key); err != nil {
				b.mu.Unlock()
				return nil, err
			}
		}
		for key := range snap.Floats {
			if err := b.ensureStatsFloat(key); err != nil {
				b.mu.Unlock()
				return nil, err
			}
		}
		logger.Info("registry bridge discovered initial stats metrics",
			zap.Int("int_keys", len(snap.Ints)),
			zap.Int("float_keys", len(snap.Floats)),
		)
	} else {
		logger.Info("registry bridge: stats registry is nil")
	}

	if inputsRegistry != nil {
		snapshot := monitoring.CollectStructSnapshot(inputsRegistry, monitoring.Full, false)
		for _, entry := range snapshot {
			data, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			for field, v := range data {
				switch v.(type) {
				case float64:
					if err := b.ensureInputFloat(field); err != nil {
						b.mu.Unlock()
						return nil, err
					}
				case int64, uint64, int:
					if err := b.ensureInputInt(field); err != nil {
						b.mu.Unlock()
						return nil, err
					}
				}
			}
		}
	}
	b.mu.Unlock()

	instruments := b.allInstruments()
	logger.Info("registry bridge registering callback",
		zap.Int("total_instruments", len(instruments)),
		zap.Int("int_gauges", len(b.intGauges)),
		zap.Int("int_counters", len(b.intCounters)),
		zap.Int("float_gauges", len(b.floatGauges)),
		zap.Int("input_int_gauges", len(b.inputIntGauges)),
		zap.Int("input_int_counters", len(b.inputIntCounters)),
		zap.Int("input_float_gauges", len(b.inputFloatGauges)),
		zap.String("meter_provider_type", fmt.Sprintf("%T", mp)),
	)

	if err := b.registerCallback(); err != nil {
		return nil, err
	}
	return b, nil
}

// Shutdown unregisters the async callback and waits for any pending
// re-registration goroutines to complete.
func (b *RegistryBridge) Shutdown() {
	// Mark closed and unregister first so no new callbacks fire and no
	// in-flight createAndReRegister goroutine re-registers after we wait.
	b.mu.Lock()
	b.closed = true
	if b.registration != nil {
		_ = b.registration.Unregister()
		b.registration = nil
	}
	b.mu.Unlock()

	// Wait for any in-flight re-registration goroutines to drain.
	b.reRegWg.Wait()
}

// ensureStatsInt creates an int instrument for the given stats key if one does
// not already exist. System-level metrics are skipped. Gauge vs counter is
// determined by isGauge.
// Caller must hold b.mu (write lock). Must NOT be called from within an OTel
// callback (pipeline lock deadlock).
func (b *RegistryBridge) ensureStatsInt(key string) error {
	if isSystemMetric(key) {
		return nil
	}
	if _, ok := b.intGauges[key]; ok {
		return nil
	}
	if _, ok := b.intCounters[key]; ok {
		return nil
	}
	if isGauge(key) {
		inst, err := b.meter.Int64ObservableGauge(key)
		if err != nil {
			return err
		}
		b.intGauges[key] = inst
	} else {
		inst, err := b.meter.Int64ObservableCounter(key)
		if err != nil {
			return err
		}
		b.intCounters[key] = inst
	}
	return nil
}

// ensureStatsFloat creates a float gauge instrument for the given stats key.
// System-level metrics are skipped. All float metrics in beats are gauges.
// Caller must hold b.mu (write lock). Must NOT be called from within an OTel
// callback.
func (b *RegistryBridge) ensureStatsFloat(key string) error {
	if isSystemMetric(key) {
		return nil
	}
	if _, ok := b.floatGauges[key]; ok {
		return nil
	}
	inst, err := b.meter.Float64ObservableGauge(key)
	if err != nil {
		return err
	}
	b.floatGauges[key] = inst
	return nil
}

// ensureInputInt creates a per-input int instrument for the given field name.
// Caller must hold b.mu (write lock). Must NOT be called from within an OTel
// callback.
func (b *RegistryBridge) ensureInputInt(field string) error {
	if _, ok := b.inputIntGauges[field]; ok {
		return nil
	}
	if _, ok := b.inputIntCounters[field]; ok {
		return nil
	}
	if isGauge(field) {
		inst, err := b.meter.Int64ObservableGauge(field)
		if err != nil {
			return err
		}
		b.inputIntGauges[field] = inst
	} else {
		inst, err := b.meter.Int64ObservableCounter(field)
		if err != nil {
			return err
		}
		b.inputIntCounters[field] = inst
	}
	return nil
}

// ensureInputFloat creates a per-input float gauge instrument for the given
// field name. All float metrics in beats are gauges.
// Caller must hold b.mu (write lock). Must NOT be called from within an OTel
// callback.
func (b *RegistryBridge) ensureInputFloat(field string) error {
	if _, ok := b.inputFloatGauges[field]; ok {
		return nil
	}
	inst, err := b.meter.Float64ObservableGauge(field)
	if err != nil {
		return err
	}
	b.inputFloatGauges[field] = inst
	return nil
}

// registerCallback registers a single OTel async callback that covers all
// currently known instruments. At least one instrument must exist for the
// OTel SDK to actually invoke the callback — RegisterCallback with zero
// instruments returns a noop registration that never fires. Since dynamic
// metric discovery relies on the callback running, zero instruments at
// startup means no metrics will ever be bridged. In practice this never
// happens because SetupMetricsOptions and the pipeline register metrics
// before the bridge is created.
func (b *RegistryBridge) registerCallback() error {
	instruments := b.allInstruments()

	if len(instruments) == 0 {
		return fmt.Errorf("registry bridge has zero instruments; dynamic metric discovery requires at least one instrument at startup")
	}

	reg, err := b.meter.RegisterCallback(b.callback, instruments...)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.registration = reg
	b.mu.Unlock()
	return nil
}

// allInstruments returns all known instruments as a slice of metric.Observable.
func (b *RegistryBridge) allInstruments() []metric.Observable {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]metric.Observable, 0,
		len(b.intGauges)+len(b.intCounters)+len(b.floatGauges)+
			len(b.inputIntGauges)+len(b.inputIntCounters)+len(b.inputFloatGauges))
	for _, inst := range b.intGauges {
		out = append(out, inst)
	}
	for _, inst := range b.intCounters {
		out = append(out, inst)
	}
	for _, inst := range b.floatGauges {
		out = append(out, inst)
	}
	for _, inst := range b.inputIntGauges {
		out = append(out, inst)
	}
	for _, inst := range b.inputIntCounters {
		out = append(out, inst)
	}
	for _, inst := range b.inputFloatGauges {
		out = append(out, inst)
	}
	return out
}

// callback is the single OTel async callback that walks both registries.
//
// New metric keys that weren't known at construction time are queued for async
// instrument creation — the OTel SDK holds the pipeline lock during callback
// execution, so neither instrument creation nor callback re-registration can
// happen synchronously here.
func (b *RegistryBridge) callback(_ context.Context, obs metric.Observer) error {
	b.collectStats(obs)
	b.collectInputs(obs)

	b.mu.Lock()
	hasPending := !b.closed &&
		(len(b.pendingStatsInts) > 0 || len(b.pendingStatsFloats) > 0 ||
			len(b.pendingInputInts) > 0 || len(b.pendingInputFloats) > 0)
	var statsInts, statsFloats, inputInts, inputFloats []string
	if hasPending {
		statsInts = b.pendingStatsInts
		statsFloats = b.pendingStatsFloats
		inputInts = b.pendingInputInts
		inputFloats = b.pendingInputFloats
		b.pendingStatsInts = nil
		b.pendingStatsFloats = nil
		b.pendingInputInts = nil
		b.pendingInputFloats = nil
		// Add inside the lock so Shutdown's Wait() cannot return before
		// this goroutine is tracked.
		b.reRegWg.Add(1)
	}
	b.mu.Unlock()

	if hasPending {
		go func() {
			defer b.reRegWg.Done()
			b.createAndReRegister(statsInts, statsFloats, inputInts, inputFloats)
		}()
	}
	return nil
}

// createAndReRegister creates instruments for pending keys and re-registers
// the callback with the full instrument set. Runs outside the OTel callback
// so the pipeline lock is not held. Serialized by reRegMu so that
// overlapping goroutines cannot leak OTel registrations.
func (b *RegistryBridge) createAndReRegister(statsInts, statsFloats, inputInts, inputFloats []string) {
	b.reRegMu.Lock()
	defer b.reRegMu.Unlock()

	// Batch-create all pending instruments and unregister the old callback
	// under a single write lock, reducing lock/unlock cycles.
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}

	for _, key := range statsInts {
		if err := b.ensureStatsInt(key); err != nil {
			b.logger.Warn("failed to create stats int instrument", zap.String("key", key), zap.Error(err))
		}
	}
	for _, key := range statsFloats {
		if err := b.ensureStatsFloat(key); err != nil {
			b.logger.Warn("failed to create stats float instrument", zap.String("key", key), zap.Error(err))
		}
	}
	for _, field := range inputInts {
		if err := b.ensureInputInt(field); err != nil {
			b.logger.Warn("failed to create input int instrument", zap.String("field", field), zap.Error(err))
		}
	}
	for _, field := range inputFloats {
		if err := b.ensureInputFloat(field); err != nil {
			b.logger.Warn("failed to create input float instrument", zap.String("field", field), zap.Error(err))
		}
	}

	if b.registration != nil {
		_ = b.registration.Unregister()
		b.registration = nil
	}
	b.mu.Unlock()

	// allInstruments() takes its own RLock, so we must not hold the write
	// lock here. RegisterCallback is also called without the lock held.
	instruments := b.allInstruments()
	reg, err := b.meter.RegisterCallback(b.callback, instruments...)
	if err != nil {
		b.logger.Error("failed to re-register OTel callback", zap.Error(err))
		return
	}
	b.logger.Info("registry bridge re-registered callback", zap.Int("instruments", len(instruments)))

	b.mu.Lock()
	if b.closed {
		_ = reg.Unregister()
	} else {
		b.registration = reg
	}
	b.mu.Unlock()
}

// collectStats takes a flat snapshot and observes all numeric values.
func (b *RegistryBridge) collectStats(obs metric.Observer) {
	if b.statsRegistry == nil {
		return
	}
	snap := monitoring.CollectFlatSnapshot(b.statsRegistry, monitoring.Full, false)

	var newInts, newFloats []string

	b.mu.RLock()
	for key, value := range snap.Ints {
		if isSystemMetric(key) {
			continue
		}
		if inst, ok := b.intGauges[key]; ok {
			obs.ObserveInt64(inst, value, b.statsAttrs)
			continue
		}
		if inst, ok := b.intCounters[key]; ok {
			obs.ObserveInt64(inst, value, b.statsAttrs)
			continue
		}
		newInts = append(newInts, key)
	}
	for key, value := range snap.Floats {
		if isSystemMetric(key) {
			continue
		}
		if inst, ok := b.floatGauges[key]; ok {
			obs.ObserveFloat64(inst, value, b.statsAttrs)
			continue
		}
		newFloats = append(newFloats, key)
	}
	b.mu.RUnlock()

	if len(newInts) > 0 || len(newFloats) > 0 {
		b.mu.Lock()
		b.pendingStatsInts = append(b.pendingStatsInts, newInts...)
		b.pendingStatsFloats = append(b.pendingStatsFloats, newFloats...)
		b.mu.Unlock()
	}
}

// collectInputs walks the inputs registry and reports per-input metrics.
func (b *RegistryBridge) collectInputs(obs metric.Observer) {
	if b.inputsRegistry == nil {
		return
	}
	snapshot := monitoring.CollectStructSnapshot(b.inputsRegistry, monitoring.Full, false)

	var newInts, newFloats []string

	b.mu.RLock()
	for _, entry := range snapshot {
		// Each input sub-registry is collected as a map[string]interface{}.
		// Skip any unexpected non-map entries defensively.
		data, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		inputID, _ := data["id"].(string)
		inputType, _ := data["input"].(string)
		if inputID == "" || inputType == "" {
			continue
		}

		attrs := metric.WithAttributeSet(attribute.NewSet(
			attribute.String("receiver", b.receiverID),
			attribute.String("input_id", inputID),
			attribute.String("input_type", inputType),
		))

		for field, v := range data {
			if fval, ok := v.(float64); ok {
				if inst, found := b.inputFloatGauges[field]; found {
					obs.ObserveFloat64(inst, fval, attrs)
					continue
				}
				newFloats = append(newFloats, field)
				continue
			}

			val, ok := toInt64Value(v)
			if !ok {
				continue
			}
			if inst, found := b.inputIntGauges[field]; found {
				obs.ObserveInt64(inst, val, attrs)
				continue
			}
			if inst, found := b.inputIntCounters[field]; found {
				obs.ObserveInt64(inst, val, attrs)
				continue
			}
			newInts = append(newInts, field)
		}
	}
	b.mu.RUnlock()

	if len(newInts) > 0 || len(newFloats) > 0 {
		b.mu.Lock()
		b.pendingInputInts = append(b.pendingInputInts, newInts...)
		b.pendingInputFloats = append(b.pendingInputFloats, newFloats...)
		b.mu.Unlock()
	}
}

// toInt64Value converts a monitoring snapshot value to int64. Returns false for
// non-numeric types.
func toInt64Value(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case uint64:
		if n > math.MaxInt64 {
			return math.MaxInt64, true
		}
		return int64(n), true // #nosec G115 — clamped above
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

// systemMetricPrefixes lists key prefixes for metrics that describe the host
// or process rather than receiver-specific work. These are excluded from the
// per-receiver bridge to avoid identical values being reported by every
// receiver in the same process.
var systemMetricPrefixes = []string{
	"beat.memstats.",
	"beat.cpu.",
	"beat.handles.",
	"beat.runtime.",
	"beat.cgroup.",
	"beat.info.uptime.ms",
	"system.",
}

// isSystemMetric returns true for metrics that describe the host or process
// (memory, CPU, load, handles, goroutines, cgroup, uptime). These are the same
// regardless of which receiver reports them.
func isSystemMetric(key string) bool {
	for _, prefix := range systemMetricPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// isGauge returns true when the given metric key represents a gauge value.
// It delegates to the log reporter's IsGauge, checking both the raw key and
// the "libbeat."-prefixed form since the log reporter's gauge set uses that
// prefix for pipeline/output/config metrics while statsRegistry keys omit it.
func isGauge(key string) bool {
	return logreport.IsGauge(key) || logreport.IsGauge("libbeat."+key)
}

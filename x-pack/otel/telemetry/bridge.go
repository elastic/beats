// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"context"
	"sync"

	logreport "github.com/elastic/beats/v7/libbeat/monitoring/report/log"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"
)

const scopeName = "github.com/elastic/beats/v7/x-pack/otel/telemetry"

// RegistryBridge dynamically discovers metrics from beats monitoring registries
// and bridges them into OTel async instruments. Instruments are auto-created
// from FlatSnapshot/StructSnapshot keys — no hardcoded metric mappings.
type RegistryBridge struct {
	logger         *zap.Logger
	meter          metric.Meter
	statsRegistry  *monitoring.Registry
	inputsRegistry *monitoring.Registry

	mu           sync.Mutex
	registration metric.Registration
	reRegWg      sync.WaitGroup

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
func NewRegistryBridge(settings component.TelemetrySettings, statsRegistry, inputsRegistry *monitoring.Registry) (*RegistryBridge, error) {
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
		statsRegistry:    statsRegistry,
		inputsRegistry:   inputsRegistry,
		intGauges:        make(map[string]metric.Int64ObservableGauge),
		intCounters:      make(map[string]metric.Int64ObservableCounter),
		floatGauges:      make(map[string]metric.Float64ObservableGauge),
		inputIntGauges:   make(map[string]metric.Int64ObservableGauge),
		inputIntCounters: make(map[string]metric.Int64ObservableCounter),
		inputFloatGauges: make(map[string]metric.Float64ObservableGauge),
	}

	// Discover initial stats metrics. Only ints and floats are bridged, because
	// OTel instruments are only numeric, all other types are skipped.
	if statsRegistry != nil {
		snap := monitoring.CollectFlatSnapshot(statsRegistry, monitoring.Full, false)
		for key := range snap.Ints {
			if err := b.ensureStatsInt(key); err != nil {
				return nil, err
			}
		}
		for key := range snap.Floats {
			if err := b.ensureStatsFloat(key); err != nil {
				return nil, err
			}
		}
	}

	// Discover initial per-input metrics.
	if inputsRegistry != nil {
		snapshot := monitoring.CollectStructSnapshot(inputsRegistry, monitoring.Full, false)
		for _, entry := range snapshot {
			// Each input sub-registry is collected as a map[string]interface{}.
			// Skip any unexpected non-map entries.
			data, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			for field, v := range data {
				switch v.(type) {
				case float64:
					if err := b.ensureInputFloat(field); err != nil {
						return nil, err
					}
				case int64, uint64, int:
					if err := b.ensureInputInt(field); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	if err := b.registerCallback(); err != nil {
		return nil, err
	}
	return b, nil
}

// Shutdown unregisters the async callback and waits for any pending
// re-registration to complete.
func (b *RegistryBridge) Shutdown() {
	b.reRegWg.Wait()
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.registration != nil {
		b.registration.Unregister()
		b.registration = nil
	}
}

// ensureStatsInt creates an int instrument for the given stats key if one does
// not already exist. Gauge vs counter is determined by isGauge.
// Must NOT be called from within an OTel callback (pipeline lock deadlock).
func (b *RegistryBridge) ensureStatsInt(key string) error {
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
// All float metrics in beats are gauges.
// Must NOT be called from within an OTel callback.
func (b *RegistryBridge) ensureStatsFloat(key string) error {
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
// Must NOT be called from within an OTel callback.
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
// Must NOT be called from within an OTel callback.
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
// currently known instruments.
func (b *RegistryBridge) registerCallback() error {
	instruments := b.allInstruments()

	var reg metric.Registration
	var err error
	if len(instruments) == 0 {
		reg, err = b.meter.RegisterCallback(b.callback)
	} else {
		reg, err = b.meter.RegisterCallback(b.callback, instruments...)
	}
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
	hasPending := len(b.pendingStatsInts) > 0 || len(b.pendingStatsFloats) > 0 ||
		len(b.pendingInputInts) > 0 || len(b.pendingInputFloats) > 0
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
	}
	b.mu.Unlock()

	if hasPending {
		b.reRegWg.Add(1)
		go func() {
			defer b.reRegWg.Done()
			b.createAndReRegister(statsInts, statsFloats, inputInts, inputFloats)
		}()
	}
	return nil
}

// createAndReRegister creates instruments for pending keys and re-registers
// the callback with the full instrument set. Runs outside the OTel callback
// so the pipeline lock is not held.
func (b *RegistryBridge) createAndReRegister(statsInts, statsFloats, inputInts, inputFloats []string) {
	// These errors should not occur in practice — the OTel SDK only fails
	// instrument creation or callback registration if the meter provider is
	// already shut down. Log for debuggability just in case.
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

	// Unregister old callback and register new one with full instrument set.
	b.mu.Lock()
	if b.registration != nil {
		b.registration.Unregister()
		b.registration = nil
	}
	b.mu.Unlock()

	instruments := b.allInstruments()
	var reg metric.Registration
	var err error
	if len(instruments) == 0 {
		reg, err = b.meter.RegisterCallback(b.callback)
	} else {
		reg, err = b.meter.RegisterCallback(b.callback, instruments...)
	}
	// Should not happen in practice — see comment above.
	if err != nil {
		b.logger.Error("failed to re-register OTel callback", zap.Error(err))
		return
	}
	b.mu.Lock()
	b.registration = reg
	b.mu.Unlock()
}

// collectStats takes a flat snapshot and observes all numeric values.
func (b *RegistryBridge) collectStats(obs metric.Observer) {
	if b.statsRegistry == nil {
		return
	}
	snap := monitoring.CollectFlatSnapshot(b.statsRegistry, monitoring.Full, false)

	for key, value := range snap.Ints {
		if inst, ok := b.intGauges[key]; ok {
			obs.ObserveInt64(inst, value)
			continue
		}
		if inst, ok := b.intCounters[key]; ok {
			obs.ObserveInt64(inst, value)
			continue
		}
		// New key discovered — queue for async instrument creation.
		b.mu.Lock()
		b.pendingStatsInts = append(b.pendingStatsInts, key)
		b.mu.Unlock()
	}

	for key, value := range snap.Floats {
		if inst, ok := b.floatGauges[key]; ok {
			obs.ObserveFloat64(inst, value)
			continue
		}
		b.mu.Lock()
		b.pendingStatsFloats = append(b.pendingStatsFloats, key)
		b.mu.Unlock()
	}
}

// collectInputs walks the inputs registry and reports per-input metrics.
func (b *RegistryBridge) collectInputs(obs metric.Observer) {
	if b.inputsRegistry == nil {
		return
	}
	snapshot := monitoring.CollectStructSnapshot(b.inputsRegistry, monitoring.Full, false)
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
			attribute.String("input_id", inputID),
			attribute.String("input_type", inputType),
		))

		for field, v := range data {
			if fval, ok := v.(float64); ok {
				if inst, found := b.inputFloatGauges[field]; found {
					obs.ObserveFloat64(inst, fval, attrs)
					continue
				}
				b.mu.Lock()
				b.pendingInputFloats = append(b.pendingInputFloats, field)
				b.mu.Unlock()
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
			b.mu.Lock()
			b.pendingInputInts = append(b.pendingInputInts, field)
			b.mu.Unlock()
		}
	}
}

// toInt64Value converts a monitoring snapshot value to int64. Returns false for
// non-numeric types.
func toInt64Value(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case uint64:
		return int64(n), true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

// isGauge returns true when the given metric key represents a gauge value.
// It delegates to the log reporter's IsGauge, checking both the raw key and
// the "libbeat."-prefixed form since the log reporter's gauge set uses that
// prefix for pipeline/output/config metrics while statsRegistry keys omit it.
func isGauge(key string) bool {
	return logreport.IsGauge(key) || logreport.IsGauge("libbeat."+key)
}

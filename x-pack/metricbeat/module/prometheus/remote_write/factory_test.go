// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"maps"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	rw "github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
)

func newFactoryBaseMetricSet(t *testing.T, overrides map[string]any) mb.BaseMetricSet {
	t.Helper()
	raw := map[string]any{
		"module":     "prometheus",
		"metricsets": []string{"remote_write"},
		"use_types":  true,
		"period":     "60s",
	}
	maps.Copy(raw, overrides)
	c, err := conf.NewConfigFrom(raw)
	require.NoError(t, err)
	_, bases, err := mb.NewModule(c, mb.Registry, beat.Info{
		Paths:  paths.New(),
		Logger: logptest.NewTestingLogger(t, ""),
	})
	require.NoError(t, err)
	require.Len(t, bases, 1)
	ms, ok := bases[0].(*rw.MetricSet)
	require.True(t, ok)
	return ms.BaseMetricSet
}

func TestRemoteWriteEventsGeneratorFactoryAppliesHistogramAssemblyDefaults(t *testing.T) {
	base := newFactoryBaseMetricSet(t, map[string]any{
		"histogram_assembly": map[string]any{
			"enabled": true,
		},
	})
	gen, err := remoteWriteEventsGeneratorFactory(base)
	require.NoError(t, err, "factory must validate and apply histogram_assembly defaults after unpack")

	typed, ok := gen.(*remoteWriteTypedGenerator)
	require.True(t, ok)
	assert.True(t, rw.RequiresRemoteWriteOwnerLoop(typed), "enabled histogram assembler must opt into owner-loop processing")
	assert.Equal(t, 5*time.Second, typed.assemblyConfig.QuietPeriod)
	assert.Equal(t, 30*time.Second, typed.assemblyConfig.HardTimeout)
	assert.Equal(t, 10_000, typed.assemblyConfig.MaxPendingHistograms)
	assert.Equal(t, 100_000, typed.assemblyConfig.MaxPendingBuckets)

	ts := model.TimeFromUnix(100)
	err = typed.CheckCapacity(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 1, ts),
	})
	require.NoError(t, err, "normal bucket batch must pass capacity after defaults are applied")
}

func TestRemoteWriteEventsGeneratorFactoryRejectsInvalidHistogramAssembly(t *testing.T) {
	mod := mbtest.NewTestModule(t, map[string]any{
		"use_types": true,
		"period":    "60s",
		"histogram_assembly": map[string]any{
			"enabled":      true,
			"quiet_period": "31s",
			"hard_timeout": "30s",
		},
	})
	_, err := loadRemoteWriteModuleConfig(mod)
	require.Error(t, err, "invalid histogram_assembly must fail config load used by factory")
	assert.Contains(t, err.Error(), "quiet_period")
}

func TestRemoteWriteEventsGeneratorFactoryDisabledHistogramAssemblyIgnoresInvalidTuning(t *testing.T) {
	base := newFactoryBaseMetricSet(t, map[string]any{
		"use_types": false,
		"histogram_assembly": map[string]any{
			"quiet_period": "-1s",
		},
	})
	gen, err := remoteWriteEventsGeneratorFactory(base)
	require.NoError(t, err)
	_, ok := gen.(*remoteWriteTypedGenerator)
	require.False(t, ok, "use_types=false must not construct typed generator")
	_, ok = gen.(*rw.RemoteWriteEventGenerator)
	require.True(t, ok, "use_types=false must use OSS generator")
}

func TestRemoteWriteEventsGeneratorFactoryDefaultsToRequestLocalHistogramConversion(t *testing.T) {
	base := newFactoryBaseMetricSet(t, nil)
	gen, err := remoteWriteEventsGeneratorFactory(base)
	require.NoError(t, err)

	typed, ok := gen.(*remoteWriteTypedGenerator)
	require.True(t, ok, "use_types=true must construct the typed generator")
	assert.Nil(t, typed.assembler, "histogram assembler must be disabled by default")
	assert.False(t, rw.RequiresRemoteWriteOwnerLoop(typed), "assembler-disabled typed generator must use the events-channel flow")
	assert.Zero(t, typed.NextFlushInterval(), "disabled assembler must not start the owner-loop flush ticker")

	timestamp := model.TimeFromUnix(100)
	events := typed.GenerateEvents(model.Samples{
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.25"}, 10, timestamp),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "0.50"}, 20, timestamp),
		promBucketSample("http_request_duration_seconds_bucket", map[string]string{"runtime": "linux", "le": "+Inf"}, 30, timestamp),
	})

	require.Len(t, events, 1, "request-local histogram conversion must emit a complete histogram in the request")
	for _, event := range events {
		assert.Contains(t, event.ModuleFields, "http_request_duration_seconds", "request-local histogram conversion must emit the histogram immediately")
	}
}

func TestMetricSetFlowMatchesHistogramAssemblerConfiguration(t *testing.T) {
	tests := []struct {
		name               string
		useAssembler       bool
		wantOwnerLoop      bool
		wantEventsChannel  bool
		wantBatchesChannel bool
	}{
		{
			name:              "assembler disabled",
			wantEventsChannel: true,
		},
		{
			name:               "assembler enabled",
			useAssembler:       true,
			wantOwnerLoop:      true,
			wantBatchesChannel: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := map[string]any{
				"module":     "prometheus",
				"metricsets": []string{"remote_write"},
				"use_types":  true,
				"period":     "60s",
				"histogram_assembly": map[string]any{
					"enabled": test.useAssembler,
				},
			}
			ms := mbtest.NewMetricSet(t, config)
			m, ok := ms.(*rw.MetricSet)
			require.True(t, ok, "expected OSS remote_write MetricSet, got %T", ms)

			useOwnerLoop, hasEvents, hasBatches := m.FlowModeForTest()
			assert.Equal(t, test.wantOwnerLoop, useOwnerLoop, "owner-loop mode must match assembler configuration")
			assert.Equal(t, test.wantEventsChannel, hasEvents, "events channel allocation must match flow mode")
			assert.Equal(t, test.wantBatchesChannel, hasBatches, "batches channel allocation must match flow mode")
		})
	}
}

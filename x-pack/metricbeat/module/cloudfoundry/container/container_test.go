// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package container

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudfoundry/sonde-go/events"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/cloudfoundry/mtest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	if err := mb.Registry.AddModule("cloudfoundrytest", mtest.NewModuleMock); err != nil {
		panic(err)
	}
	mb.Registry.MustAddMetricSet("cloudfoundrytest", "test", newTestMetricSet,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

func newTestMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return New(base)
}

func TestMetricSet(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("cloudfoundry"))

	config := map[string]interface{}{
		"module":        "cloudfoundrytest",
		"client_id":     "dummy",
		"client_secret": "dummy",
		"api_address":   "dummy",
		"shard_id":      "dummy",
	}

	ms := mbtest.NewPushMetricSetV2(t, config)
	hub := ms.Module().(*mtest.ModuleMock).Hub

	go func() {
		hub.SendEnvelope(containerMetricsEnvelope(containerMetrics{app: "1234", memory: 1024, cpupct: 12.34}))
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 1, ms)
	require.NotEmpty(t, events)

	expectedFields := mapstr.M{
		"cloudfoundry.app.id":                       "1234",
		"cloudfoundry.container.cpu.pct":            float64(0.1234),
		"cloudfoundry.container.disk.bytes":         uint64(0),
		"cloudfoundry.container.disk.quota.bytes":   uint64(0),
		"cloudfoundry.container.instance_index":     int32(0),
		"cloudfoundry.container.memory.bytes":       uint64(1024),
		"cloudfoundry.container.memory.quota.bytes": uint64(0),
		"cloudfoundry.envelope.deployment":          "test",
		"cloudfoundry.envelope.index":               "index",
		"cloudfoundry.envelope.ip":                  "127.0.0.1",
		"cloudfoundry.envelope.job":                 "test",
		"cloudfoundry.envelope.origin":              "test",
		"cloudfoundry.type":                         "container",
	}
	require.Equal(t, expectedFields, events[0].RootFields.Flatten())
}

func TestMetricValuesAreNumbers(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("cloudfoundry"))

	config := map[string]interface{}{
		"module":        "cloudfoundrytest",
		"client_id":     "dummy",
		"client_secret": "dummy",
		"api_address":   "dummy",
		"shard_id":      "dummy",
	}

	ms := mbtest.NewPushMetricSetV2(t, config)
	hub := ms.Module().(*mtest.ModuleMock).Hub

	go func() {
		hub.SendEnvelope(containerMetricsEnvelope(containerMetrics{app: "0000", memory: 1024, cpupct: math.NaN()}))
		hub.SendEnvelope(containerMetricsEnvelope(containerMetrics{app: "1234", memory: 1024, cpupct: 12.34}))
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 2, ms)
	require.NotEmpty(t, events)

	for _, e := range events {
		memory, err := e.RootFields.GetValue("cloudfoundry.container.memory.bytes")
		if assert.NoError(t, err, "checking memory") {
			assert.Equal(t, uint64(1024), memory.(uint64))
		}

		app, err := e.RootFields.GetValue("cloudfoundry.app.id")
		require.NoError(t, err, "getting app id")

		cpuPctKey := "cloudfoundry.container.cpu.pct"
		switch app {
		case "0000":
			_, err := e.RootFields.GetValue(cpuPctKey)
			require.Error(t, err, "non-numeric metric shouldn't be there")
		case "1234":
			v, err := e.RootFields.GetValue(cpuPctKey)
			if assert.NoError(t, err, "checking cpu pct") {
				assert.Equal(t, 0.1234, v.(float64))
			}
		default:
			t.Errorf("unexpected app: %s", app)
		}
	}
}

type containerMetrics struct {
	app         string
	instance    int32
	cpupct      float64
	memory      uint64
	disk        uint64
	memoryQuota uint64
	diskQuota   uint64
}

func containerMetricsEnvelope(metrics containerMetrics) *events.Envelope {
	eventType := events.Envelope_ContainerMetric
	origin := "test"
	deployment := "test"
	job := "test"
	ip := "127.0.0.1"
	index := "index"
	timestamp := time.Now().Unix()
	return &events.Envelope{
		EventType:  &eventType,
		Timestamp:  &timestamp,
		Origin:     &origin,
		Deployment: &deployment,
		Job:        &job,
		Ip:         &ip,
		Index:      &index,
		ContainerMetric: &events.ContainerMetric{
			ApplicationId:    &metrics.app,
			InstanceIndex:    &metrics.instance,
			CpuPercentage:    &metrics.cpupct,
			MemoryBytes:      &metrics.memory,
			DiskBytes:        &metrics.disk,
			MemoryBytesQuota: &metrics.memoryQuota,
			DiskBytesQuota:   &metrics.diskQuota,
		},
	}
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package value

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudfoundry/sonde-go/events"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/cloudfoundry/mtest"
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
		hub.SendEnvelope(valueMetricEnvelope("duration", 12.34, "ms"))
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 1, ms)
	require.NotEmpty(t, events)

	expectedFields := common.MapStr{
		"cloudfoundry.envelope.deployment": "test",
		"cloudfoundry.envelope.index":      "index",
		"cloudfoundry.envelope.ip":         "127.0.0.1",
		"cloudfoundry.envelope.job":        "test",
		"cloudfoundry.envelope.origin":     "test",
		"cloudfoundry.type":                "value",
		"cloudfoundry.value.name":          "duration",
		"cloudfoundry.value.unit":          "ms",
		"cloudfoundry.value.value":         float64(12.34),
	}
	require.Equal(t, expectedFields, events[0].RootFields.Flatten())
}

func TestValuesAreNumbers(t *testing.T) {
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
		hub.SendEnvelope(valueMetricEnvelope("duration", math.NaN(), "ms"))
		hub.SendEnvelope(valueMetricEnvelope("duration", 12.34, "ms"))
		hub.SendEnvelope(valueMetricEnvelope("duration", math.Inf(1), "ms"))
		hub.SendEnvelope(valueMetricEnvelope("duration", 34.56, "ms"))
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 2, ms)
	require.NotEmpty(t, events)

	for _, e := range events {
		value, err := e.RootFields.GetValue("cloudfoundry.value.value")
		if assert.NoError(t, err) {
			assert.False(t, math.IsNaN(value.(float64)))
			assert.False(t, math.IsInf(value.(float64), 0))
		}
	}
}

func valueMetricEnvelope(name string, value float64, unit string) *events.Envelope {
	eventType := events.Envelope_ValueMetric
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
		ValueMetric: &events.ValueMetric{
			Name:  &name,
			Value: &value,
			Unit:  &unit,
		},
	}
}

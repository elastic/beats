// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package counter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cloudfoundry/sonde-go/events"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/cloudfoundry/mtest"
	"github.com/elastic/elastic-agent-libs/logp"
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
		hub.SendEnvelope(counterMetricEnvelope("requests", 1234, 123))
	}()

	events := mbtest.RunPushMetricSetV2(10*time.Second, 1, ms)
	require.NotEmpty(t, events)

	expectedFields := mapstr.M{
		"cloudfoundry.counter.delta":       uint64(123),
		"cloudfoundry.counter.name":        "requests",
		"cloudfoundry.counter.total":       uint64(1234),
		"cloudfoundry.envelope.deployment": "test",
		"cloudfoundry.envelope.index":      "index",
		"cloudfoundry.envelope.ip":         "127.0.0.1",
		"cloudfoundry.envelope.job":        "test",
		"cloudfoundry.envelope.origin":     "test",
		"cloudfoundry.type":                "counter",
	}
	require.Equal(t, expectedFields, events[0].RootFields.Flatten())
}

func counterMetricEnvelope(name string, total uint64, delta uint64) *events.Envelope {
	eventType := events.Envelope_CounterEvent
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
		CounterEvent: &events.CounterEvent{
			Name:  &name,
			Total: &total,
			Delta: &delta,
		},
	}
}

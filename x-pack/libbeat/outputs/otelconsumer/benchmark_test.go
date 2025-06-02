// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelconsumer

import (
	"context"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

func makeTestOtelConsumer(t testing.TB, consumeFn func(ctx context.Context, ld plog.Logs) error) *otelConsumer {
	t.Helper()

	logConsumer, err := consumer.NewLogs(consumeFn)
	assert.NoError(t, err)
	consumer := &otelConsumer{
		observer:     outputs.NewNilObserver(),
		logsConsumer: logConsumer,
		beatInfo:     beat.Info{},
		log:          logp.NewLogger("otelconsumer"),
	}
	return consumer
}

func BenchmarkPublish(b *testing.B) {
	events := make([]beat.Event, 0, b.N)
	for i := 0; i < b.N; i++ {
		events = append(events, beat.Event{Fields: mapstr.M{"field": i}})
	}
	batch := outest.NewBatch(events...)
	var countLogs int
	otelConsumer := makeTestOtelConsumer(b, func(ctx context.Context, ld plog.Logs) error {
		countLogs = countLogs + ld.LogRecordCount()
		return nil
	})

	err := otelConsumer.Publish(context.Background(), batch)
	assert.NoError(b, err)
	assert.Len(b, batch.Signals, 1)
	assert.Equal(b, outest.BatchACK, batch.Signals[0].Tag)
	assert.Equal(b, len(batch.Events()), countLogs, "all events should be consumed")
}

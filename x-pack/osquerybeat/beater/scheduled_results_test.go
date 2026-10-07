// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type scheduledResultClient struct{}

func (scheduledResultClient) ResolveResult(_ context.Context, _ string, hits []map[string]string) ([]map[string]any, error) {
	result := make([]map[string]any, len(hits))
	for i, hit := range hits {
		result[i] = make(map[string]any)
		for key, value := range hit {
			result[i][key] = value
		}
	}
	return result, nil
}

func (scheduledResultClient) Query(context.Context, string, time.Duration) ([]map[string]any, error) {
	return nil, nil
}

type stalledResponsePublisher struct {
	mockBeatPublisher
	entered chan struct{}
	release chan struct{}
}

func (p *stalledResponsePublisher) PublishScheduledResponse(string, string, string, string, string, string, time.Time, time.Time, time.Time, int, int64) {
	close(p.entered)
	<-p.release
}

func scheduledTestConfig(t *testing.T) *ConfigPlugin {
	t.Helper()
	p := NewConfigPlugin(logptest.NewTestingLogger(t, "scheduled_results"))
	disabled := false
	require.NoError(t, p.Set([]config.InputConfig{{Osquery: &config.OsqueryConfig{
		Schedule: map[string]config.Query{"test": {Query: "SELECT 1", Profiling: &disabled}},
	}}}), "test schedule must be valid")
	_, err := p.GenerateConfig(t.Context())
	require.NoError(t, err, "test schedule metadata must be applied")
	return p
}

func TestScheduledLoggerReturnsWhileOutputBlocked(t *testing.T) {
	for _, payload := range []string{
		`{"name":"test","action":"snapshot","snapshot":[],"unixTime":1704067200}`,
		`{"name":"test","action":"snapshot","snapshot":[{"value":"1"}],"unixTime":1704067200}`,
	} {
		t.Run(payload, func(t *testing.T) {
			p := &stalledResponsePublisher{entered: make(chan struct{}), release: make(chan struct{})}
			bt := &osquerybeat{pub: p, log: logptest.NewTestingLogger(t, "scheduled_results")}
			plugin, stop := bt.newScheduledLogger(t.Context(), scheduledResultClient{}, scheduledTestConfig(t))
			defer stop()
			defer close(p.release)
			done := make(chan error, 1)
			go func() { done <- plugin.Log(t.Context(), logger.LogTypeSnapshot, payload) }()
			select {
			case <-p.entered:
			case <-time.After(time.Second):
				t.Fatal("scheduled response was not attempted, including for zero hits")
			}
			select {
			case err := <-done:
				assert.NoError(t, err, "output backpressure must not fail the osquery logger")
			case <-time.After(200 * time.Millisecond):
				t.Error("logger callback waited for the unavailable output")
			}
		})
	}
}

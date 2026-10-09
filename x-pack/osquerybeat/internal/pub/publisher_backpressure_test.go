// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type publisherTestPipeline struct{ configs []beat.ClientConfig }

func (p *publisherTestPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	p.configs = append(p.configs, cfg)
	return &recordingClient{}, nil
}
func (p *publisherTestPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}
func (*publisherTestPipeline) Disconnect(context.Context) error { return nil }

func TestPublisherAllStreamsUseNonblockingAdmission(t *testing.T) {
	pipeline := &publisherTestPipeline{}
	p := New(&beat.Beat{Publisher: pipeline}, logptest.NewTestingLogger(t, "publisher"))
	defer p.Close()
	inputs := []config.InputConfig{
		{Datastream: config.DatastreamConfig{Dataset: config.DefaultDataset}},
		{Datastream: config.DatastreamConfig{Dataset: config.DefaultActionResponsesDataset}},
		{Datastream: config.DatastreamConfig{Dataset: config.DefaultQueryProfileDataset}},
	}
	require.NoError(t, p.Configure(inputs), "all publisher streams must be configured")
	require.Len(t, pipeline.configs, 3, "results, responses and profiles must each have a client")
	for _, cfg := range pipeline.configs {
		assert.Equal(t, beat.DropIfFull, cfg.PublishMode, "no osquery stream may block on output queue admission")
	}
}

func TestPublisherAdmissionLossIsObservable(t *testing.T) {
	pipeline := &publisherTestPipeline{}
	p := New(&beat.Beat{Publisher: pipeline}, logptest.NewTestingLogger(t, "publisher"))
	defer p.Close()
	require.NoError(t, p.Configure([]config.InputConfig{
		{Datastream: config.DatastreamConfig{Dataset: config.DefaultDataset}},
		{Datastream: config.DatastreamConfig{Dataset: config.DefaultActionResponsesDataset}},
		{Datastream: config.DatastreamConfig{Dataset: config.DefaultQueryProfileDataset}},
	}), "all publisher streams must be configured")
	for i, stream := range []string{config.DefaultDataset, config.DefaultActionResponsesDataset, config.DefaultQueryProfileDataset} {
		listener := pipeline.configs[i].ClientListener
		require.NotNil(t, listener, "queue admission outcomes must be observed for %s", stream)
		listener.Published()
		listener.DroppedOnPublish(beat.Event{})
		listener.DroppedOnPublish(beat.Event{})
		assert.Equal(t, uint64(1), p.listeners[stream].published.Get(), "accepted events must be counted")
		assert.Equal(t, uint64(2), p.listeners[stream].dropped.Get(), "rejected events must be counted separately")
	}
}

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build cloudfoundry

package cloudfoundry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	cftest "github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry/test"
)

func TestInput(t *testing.T) {
	config := common.MustNewConfigFrom(cftest.GetConfigFromEnv(t))

	events := make(chan beat.Event)
	connector := channel.ConnectorFunc(func(*common.Config, beat.ClientConfig) (channel.Outleter, error) {
		return newOutleter(events), nil
	})

	inputCtx := input.Context{Done: make(chan struct{})}

	input, err := NewInput(config, connector, inputCtx)
	require.NoError(t, err)

	go input.Run()
	defer input.Stop()

	select {
	case e := <-events:
		t.Logf("Event received: %+v", e)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for events")
	}
}

type outleter struct {
	events chan<- beat.Event
	done   chan struct{}
}

func newOutleter(events chan<- beat.Event) *outleter {
	return &outleter{
		events: events,
		done:   make(chan struct{}),
	}
}

func (o *outleter) Close() error {
	close(o.done)
	return nil
}

func (o *outleter) Done() <-chan struct{} {
	return o.done
}

func (o *outleter) OnEvent(e beat.Event) bool {
	select {
	case o.events <- e:
		return true
	default:
		return false
	}
}

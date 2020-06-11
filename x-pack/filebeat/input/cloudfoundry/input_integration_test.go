// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build cloudfoundry

package cloudfoundry

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	cftest "github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry/test"
)

func TestInput(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("cloudfoundry"))

	t.Run("v1", func(t *testing.T) {
		testInput(t, "v1")
	})

	t.Run("v2", func(t *testing.T) {
		testInput(t, "v2")
	})
}

func testInput(t *testing.T, version string) {
	config := common.MustNewConfigFrom(cftest.GetConfigFromEnv(t))
	config.SetString("version", -1, version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiAddress, err := config.String("api_address", -1)
	require.NoError(t, err)

	// Ensure that there is something happening in the firehose
	go makeApiRequests(t, ctx, apiAddress)

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

func makeApiRequests(t *testing.T, ctx context.Context, address string) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}

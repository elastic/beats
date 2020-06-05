// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build cloudfoundry

package cloudfoundry

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestInput(t *testing.T) {
	config := common.MustNewConfigFrom(GetConfig(t))

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

func GetConfig(t *testing.T) map[string]interface{} {
	t.Helper()

	config := map[string]interface{}{
		"api_address":   lookupEnv(t, "CLOUDFOUNDRY_API_ADDRESS"),
		"client_id":     lookupEnv(t, "CLOUDFOUNDRY_CLIENT_ID"),
		"client_secret": lookupEnv(t, "CLOUDFOUNDRY_CLIENT_SECRET"),

		"ssl.verification_mode": "none",
	}

	optionalConfig(config, "uaa_address", "CLOUDFOUNDRY_UAA_ADDRESS")
	optionalConfig(config, "rlp_address", "CLOUDFOUNDRY_RLP_ADDRESS")
	optionalConfig(config, "doppler_address", "CLOUDFOUNDRY_DOPPLER_ADDRESS")

	if t.Failed() {
		t.FailNow()
	}

	return config
}

func lookupEnv(t *testing.T, name string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		t.Errorf("Environment variable %s is not set", name)
	}
	return value
}

func optionalConfig(config map[string]interface{}, key string, envVar string) {
	if value, ok := os.LookupEnv(envVar); ok {
		config[key] = value
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

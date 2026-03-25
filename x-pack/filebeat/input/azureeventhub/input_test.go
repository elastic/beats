// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestCreateWithProcessorV1FallsBackToV2(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("azureeventhub"))
	log := logp.NewLogger("azureeventhub")

	manager := &eventHubInputManager{log: log}

	config := conf.MustNewConfigFrom(map[string]interface{}{
		"eventhub":                         "test-hub",
		"connection_string":                "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test",
		"storage_account":                  "teststorage",
		"storage_account_connection_string": "DefaultEndpointsProtocol=https;AccountName=teststorage;AccountKey=secret;EndpointSuffix=core.windows.net",
		"processor_version":                "v1",
	})

	input, err := manager.Create(config)
	require.NoError(t, err)
	require.NotNil(t, input)

	_, ok := input.(*eventHubInputV2)
	assert.True(t, ok, "expected eventHubInputV2 when processor_version is v1")
}

// ackClient is a fake beat.Client that ACKs the published messages.
type fakeClient struct {
	sync.Mutex
	publishedEvents []beat.Event
}

func (c *fakeClient) Close() error { return nil }

func (c *fakeClient) Publish(event beat.Event) {
	c.Lock()
	defer c.Unlock()
	c.publishedEvents = append(c.publishedEvents, event)
}

func (c *fakeClient) PublishAll(event []beat.Event) {
	for _, e := range event {
		c.Publish(e)
	}
}

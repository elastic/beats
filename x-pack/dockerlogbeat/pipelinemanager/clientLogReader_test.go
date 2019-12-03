// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"sync"
	"testing"

	"github.com/docker/docker/daemon/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/x-pack/dockerlogbeat/pipelinemock"
	"github.com/elastic/beats/x-pack/dockerlogbeat/pipereader"
)

func TestNewClient(t *testing.T) {
	client, teardown := setupTestClient(t)
	defer teardown()

	event := testReturn(t, client)
	assert.Equal(t, event.Fields["message"], "This is a log line")
}

func setupTestClient(t *testing.T) (*pipelinemock.MockPipelineConnector, func()) {
	mockConnector := &pipelinemock.MockPipelineConnector{}
	client := createNewClient(t, mockConnector)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.ConsumePipelineAndSend()
	}()

	return mockConnector, func() {
		client.Close()
		wg.Wait()
	}
}

func createNewClient(t *testing.T, mockConnector *pipelinemock.MockPipelineConnector) *ClientLogger {
	// an example container metadata struct
	cfgObject := logger.Info{
		Config:             map[string]string{"output.elasticsearch": "localhost:9200"},
		ContainerLabels:    map[string]string{"test.label": "test"},
		ContainerID:        "3acc92989a97c415905eba090277b8a8834d087e58a95bed55450338ce0758dd",
		ContainerName:      "testContainer",
		ContainerImageName: "TestImage",
	}

	// create a new pipeline reader for use with the libbeat client
	reader, err := pipereader.NewReaderFromReadCloser(pipelinemock.CreateTestInput(t))
	require.NoError(t, err)

	client, err := newClientFromPipeline(mockConnector, reader, "aaa", cfgObject)
	require.NoError(t, err)

	return client
}

func testReturn(t *testing.T, conn *pipelinemock.MockPipelineConnector) beat.Event {
	for {
		// wait until we get our example event back
		if events := conn.GetAllEvents(); len(events) > 0 {
			assert.NotEmpty(t, events)
			return events[0]
		}
	}

}

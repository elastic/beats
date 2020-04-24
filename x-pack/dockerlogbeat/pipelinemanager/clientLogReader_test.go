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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/dockerlogbeat/pipelinemock"
	"github.com/elastic/beats/v7/x-pack/dockerlogbeat/pipereader"
)

func TestNewClient(t *testing.T) {
	logString := "This is a log line"
	cfgObject := logger.Info{
		Config:             map[string]string{"output.elasticsearch": "localhost:9200"},
		ContainerLabels:    map[string]string{"test.label": "test"},
		ContainerID:        "3acc92989a97c415905eba090277b8a8834d087e58a95bed55450338ce0758dd",
		ContainerName:      "/testContainer",
		ContainerImageName: "TestImage",
	}
	client, teardown := setupTestReader(t, logString, cfgObject)
	defer teardown()

	event := testReturn(t, client)
	assert.Equal(t, event.Fields["message"], logString)
	assert.Equal(t, event.Fields["container"].(common.MapStr)["name"], "testContainer")
}

// setupTestReader sets up the "read side" of the pipeline, spawing a goroutine to read and event and send it back to the test.
func setupTestReader(t *testing.T, logString string, containerConfig logger.Info) (*pipelinemock.MockPipelineConnector, func()) {
	mockConnector := &pipelinemock.MockPipelineConnector{}
	client := createNewClient(t, logString, mockConnector, containerConfig)

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

// createNewClient sets up the "write side" of the pipeline, creating a log event to write and send back into the test.
func createNewClient(t *testing.T, logString string, mockConnector *pipelinemock.MockPipelineConnector, containerConfig logger.Info) *ClientLogger {
	// an example container metadata struct
	cfgObject := logger.Info{
		Config:             map[string]string{"output.elasticsearch": "localhost:9200"},
		ContainerLabels:    map[string]string{"test.label": "test"},
		ContainerID:        "3acc92989a97c415905eba090277b8a8834d087e58a95bed55450338ce0758dd",
		ContainerName:      "testContainer",
		ContainerImageName: "TestImage",
	}

	// create a new pipeline reader for use with the libbeat client
	reader, err := pipereader.NewReaderFromReadCloser(pipelinemock.CreateTestInputFromLine(t, logString))
	require.NoError(t, err)

	client, err := newClientFromPipeline(mockConnector, reader, "aaa", cfgObject)
	require.NoError(t, err)

	return client
}

// testReturn waits in a loop until we get back an event
func testReturn(t *testing.T, conn *pipelinemock.MockPipelineConnector) beat.Event {
	for {
		// wait until we get our example event back
		if events := conn.GetAllEvents(); len(events) > 0 {
			assert.NotEmpty(t, events)
			return events[0]
		}
	}

}

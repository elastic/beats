// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"testing"

	"github.com/docker/docker/daemon/logger"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/x-pack/dockerlogbeat/pipelinemock"
	"github.com/elastic/beats/x-pack/dockerlogbeat/pipereader"
)

func TestNewClient(t *testing.T) {
	mockConnector := &pipelinemock.MockPipelineConnector{}
	client := createNewClient(t, mockConnector)
	// ConsumePipelineAndSent is what does the actual reading and sending.
	// After we spawn this goroutine, we wait until we get something back
	go client.ConsumePipelineAndSend()
	event := testReturnAndClose(t, mockConnector, client)
	assert.Equal(t, event.Fields["message"], "This is a log line")

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
	assert.NoError(t, err)

	client, err := newClientFromPipeline(mockConnector, reader, "aaa", cfgObject)
	assert.NoError(t, err)

	return client
}

func testReturnAndClose(t *testing.T, conn *pipelinemock.MockPipelineConnector, client *ClientLogger) beat.Event {
	defer client.Close()
	for {
		// wait until we get our example event back
		if events := conn.GetAllEvents(); len(events) > 0 {
			assert.NotEmpty(t, events)
			return events[0]
		}
	}

}

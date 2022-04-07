// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/daemon/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/daemon/logger/jsonfilelog"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/x-pack/dockerlogbeat/pipelinemock"
	"github.com/elastic/beats/v8/x-pack/dockerlogbeat/pipereader"
)

func TestConfigHosts(t *testing.T) {
	testHostEmpty := map[string]string{
		"api_key": "keykey",
	}
	_, err := NewCfgFromRaw(testHostEmpty)
	assert.Error(t, err)

	testMultiHost := map[string]string{
		"hosts": "endpoint1,endpoint2",
	}
	goodOut := []string{"endpoint1", "endpoint2"}
	cfg, err := NewCfgFromRaw(testMultiHost)
	assert.NoError(t, err)
	assert.Equal(t, goodOut, cfg.Endpoint)

}

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
func createNewClient(t *testing.T, logString string, mockConnector *pipelinemock.MockPipelineConnector, cfgObject logger.Info) *ClientLogger {

	// create a new pipeline reader for use with the libbeat client
	reader, err := pipereader.NewReaderFromReadCloser(pipelinemock.CreateTestInputFromLine(t, logString))
	require.NoError(t, err)

	info := logger.Info{
		ContainerID: "b87d3b0379f816a5f2f7070f28cc05e2f564a3fb549a67c64ec30fc5b04142ed",
		LogPath:     filepath.Join("/tmp/dockerbeattest/", strconv.FormatInt(time.Now().Unix(), 10)),
	}

	err = os.MkdirAll(filepath.Dir(info.LogPath), 0755)
	assert.NoError(t, err)
	localLog, err := jsonfilelog.New(info)
	assert.NoError(t, err)

	client, err := newClientFromPipeline(mockConnector, reader, 123, cfgObject, localLog, "test")
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

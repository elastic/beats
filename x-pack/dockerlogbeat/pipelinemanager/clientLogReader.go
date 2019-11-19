// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
	"github.com/elastic/beats/x-pack/dockerlogbeat/pipereader"
)

// ClientLogger is an instance of a pipeline logger client meant for reading from a single log stream
// There's a many-to-one relationship between clients and pipelines.
// Each container with the same config will get its own client to the same pipeline.
type ClientLogger struct {
	logFile       *pipereader.PipeReader
	client        beat.Client
	pipelineHash  string
	closer        chan struct{}
	containerMeta logger.Info
}

// newClientFromPipeline creates a new Client logger with a FIFO reader and beat client
func newClientFromPipeline(pipeline *pipeline.Pipeline, file, hashstring string, info logger.Info) (*ClientLogger, error) {
	// setup the beat client
	settings := beat.ClientConfig{
		WaitClose: 0,
	}
	settings.ACKCount = func(n int) {
		logp.Debug("Pipeline client (%s) ACKS; %v", file, n)
	}
	settings.PublishMode = beat.DefaultGuarantees
	client, err := pipeline.ConnectWith(settings)
	if err != nil {
		return nil, err
	}

	// Create the FIFO reader client from the FIPO pipe
	inputFile, err := pipereader.NewReaderFromPath(file)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening logger fifo: %q", file)
	}
	logp.Info("Created new logger for %s", file)

	return &ClientLogger{logFile: inputFile, client: client, pipelineHash: hashstring, closer: make(chan struct{}), containerMeta: info}, nil
}

// Close closes the pipeline client and reader
func (cl *ClientLogger) Close() error {
	logp.Info("Closing ClientLogger")
	cl.logFile.Close()
	return cl.client.Close()

}

// ConsumePipelineAndSend consumes events from the FIFO pipe and sends them to the pipeline client
func (cl *ClientLogger) ConsumePipelineAndSend() {
	publishWriter := make(chan logdriver.LogEntry, 500)

	go cl.publishLoop(publishWriter)
	// Clean up the reader after we're done
	defer func() {
		close(publishWriter)

	}()

	var log logdriver.LogEntry
	for {
		err := cl.logFile.ReadMessage(&log)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting message: %s\n", err)
			return
		}
		publishWriter <- log
		log.Reset()

	}
}

// publishLoop sits in a loop and waits for events to publish
// Publish() can block if there is an upstream output issue. This is a problem because if the FIFO queues that handle the docker logs fill up, plugins can no longer send logs
// A buffered channel with its own publish gives us a little more wiggle room.
func (cl *ClientLogger) publishLoop(reader chan logdriver.LogEntry) {
	for {
		entry, ok := <-reader
		if !ok {
			logp.Info("Closing publishLoop")
			return
		}

		line := strings.TrimSpace(string(entry.Line))

		cl.client.Publish(beat.Event{
			Timestamp: time.Unix(0, entry.TimeNano),
			Fields: common.MapStr{
				"message": line,
				"container": common.MapStr{
					"labels": cl.containerMeta.ContainerLabels,
					"id":     cl.containerMeta.ContainerID,
					"name":   cl.containerMeta.ContainerName,
					"image": common.MapStr{
						"name": cl.containerMeta.ContainerImageName,
					},
				},
			},
		})

	}

}

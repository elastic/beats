// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	helper "github.com/elastic/beats/libbeat/common/docker"
	"github.com/elastic/beats/libbeat/logp"
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
	logger        *logp.Logger
}

// newClientFromPipeline creates a new Client logger with a FIFO reader and beat client
func newClientFromPipeline(pipeline beat.PipelineConnector, inputFile *pipereader.PipeReader, hashstring string, info logger.Info) (*ClientLogger, error) {
	// setup the beat client
	settings := beat.ClientConfig{
		WaitClose: 0,
	}
	clientLogger := logp.NewLogger("clientLogReader")
	settings.ACKCount = func(n int) {
		clientLogger.Debugf("Pipeline client ACKS; %v", n)
	}
	settings.PublishMode = beat.DefaultGuarantees
	client, err := pipeline.ConnectWith(settings)
	if err != nil {
		return nil, err
	}

	clientLogger.Debugf("Created new logger for %s", hashstring)

	return &ClientLogger{logFile: inputFile, client: client, pipelineHash: hashstring, closer: make(chan struct{}), containerMeta: info, logger: clientLogger}, nil
}

// Close closes the pipeline client and reader
func (cl *ClientLogger) Close() error {
	cl.logger.Debug("Closing ClientLogger")
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
			cl.logger.Error(os.Stderr, "Error getting message: %s\n", err)
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
			cl.logger.Debug("Closing publishLoop")
			return
		}

		line := strings.TrimSpace(string(entry.Line))

		cl.client.Publish(beat.Event{
			Timestamp: time.Unix(0, entry.TimeNano),
			Fields: common.MapStr{
				"message": line,
				"container": common.MapStr{
					"labels": helper.DeDotLabels(cl.containerMeta.ContainerLabels, true),
					"id":     cl.containerMeta.ContainerID,
					"name":   helper.ExtractContainerName([]string{cl.containerMeta.ContainerName}),
					"image": common.MapStr{
						"name": cl.containerMeta.ContainerImageName,
					},
				},
			},
		})

	}

}

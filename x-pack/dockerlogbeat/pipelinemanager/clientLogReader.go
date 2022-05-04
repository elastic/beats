// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"

	"github.com/docker/docker/api/types/backend"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	helper "github.com/elastic/beats/v7/libbeat/common/docker"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/dockerlogbeat/pipereader"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// ClientLogger collects logs for a docker container logging to stdout and stderr, using the FIFO provided by the docker daemon.
// Each log line is written to a local log file for retrieval via "docker logs", and forwarded to the beats publisher pipeline.
// The local log storage is based on the docker json-file logger and supports the same settings. If "max-size" is not configured, we will rotate the log file every 10MB.
type ClientLogger struct {
	// pipelineHash is a hash of the libbeat publisher pipeline config
	pipelineHash uint64
	// logger is the internal error message logger
	logger *logp.Logger
	// ContainerMeta is the metadata object for the container we get from docker
	ContainerMeta logger.Info
	// ContainerECSMeta is a container metadata object appended to every event
	ContainerECSMeta mapstr.M
	// logFile is the FIFO reader that reads from the docker container stdio
	logFile *pipereader.PipeReader
	// client is the libbeat client object that sends logs upstream
	client beat.Client
	// localLog manages the local JSON logs for containers
	localLog logger.Logger
	// hostname for event metadata
	hostname string
}

// newClientFromPipeline creates a new Client logger with a FIFO reader and beat client
func newClientFromPipeline(pipeline beat.PipelineConnector, inputFile *pipereader.PipeReader, hash uint64, info logger.Info, localLog logger.Logger, hostname string) (*ClientLogger, error) {
	// setup the beat client
	settings := beat.ClientConfig{
		WaitClose: 0,
	}
	clientLogger := logp.NewLogger("clientLogReader")
	settings.ACKHandler = acker.Counting(func(n int) {
		clientLogger.Debugf("Pipeline client ACKS; %v", n)
	})
	settings.PublishMode = beat.DefaultGuarantees
	client, err := pipeline.ConnectWith(settings)
	if err != nil {
		return nil, err
	}

	clientLogger.Debugf("Created new logger for %d", hash)

	return &ClientLogger{logFile: inputFile,
		client:           client,
		pipelineHash:     hash,
		ContainerMeta:    info,
		ContainerECSMeta: constructECSContainerData(info),
		localLog:         localLog,
		logger:           clientLogger,
		hostname:         hostname}, nil
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
			if err == io.EOF {
				return
			}
			cl.logger.Errorf("Error getting message: %s\n", err)
			return
		}
		publishWriter <- log
		log.Reset()

	}
}

// constructECSContainerData creates an ES-ready MapString object with container metadata.
func constructECSContainerData(metadata logger.Info) mapstr.M {

	var containerImageName, containerImageTag string
	if idx := strings.IndexRune(metadata.ContainerImageName, ':'); idx >= 0 {
		containerImageName = string([]rune(metadata.ContainerImageName)[:idx])
		containerImageTag = string([]rune(metadata.ContainerImageName)[idx+1:])
	}

	return mapstr.M{
		"labels": helper.DeDotLabels(metadata.ContainerLabels, true),
		"id":     metadata.ContainerID,
		"name":   helper.ExtractContainerName([]string{metadata.ContainerName}),
		"image": mapstr.M{
			"name": containerImageName,
			"tag":  containerImageTag,
		},
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

		cl.localLog.Log(constructLogSpoolMsg(entry))
		line := strings.TrimSpace(string(entry.Line))

		cl.client.Publish(beat.Event{
			Timestamp: time.Unix(0, entry.TimeNano),
			Fields: mapstr.M{
				"message":   line,
				"container": cl.ContainerECSMeta,
				"host": mapstr.M{
					"name": cl.hostname,
				},
			},
		})
	}
}

func constructLogSpoolMsg(line logdriver.LogEntry) *logger.Message {
	var msg logger.Message

	msg.Line = line.Line
	msg.Source = line.Source
	msg.Timestamp = time.Unix(0, line.TimeNano)
	if line.PartialLogMetadata != nil {
		msg.PLogMetaData = &backend.PartialLogMetaData{}
		msg.PLogMetaData.ID = line.PartialLogMetadata.Id
		msg.PLogMetaData.Last = line.PartialLogMetadata.Last
		msg.PLogMetaData.Ordinal = int(line.PartialLogMetadata.Ordinal)
	}
	return &msg
}

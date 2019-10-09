// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/docker/engine/api/types/plugins/logdriver"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
	pb "github.com/gogo/protobuf/io"
	"github.com/pkg/errors"
	"github.com/tonistiigi/fifo"
)

// ClientLogger is an instance of a pipeline logger client meant for reading from a single log stream
// There's a many-to-one relationship between clients and pipelines.
// Each container with the same config will get its own client to the same pipeline.
type ClientLogger struct {
	logFile      io.ReadWriteCloser
	client       beat.Client
	pipelineHash string
}

// newClientFromPipeline creates a new Client logger with a FIFO reader and beat client
func newClientFromPipeline(pipeline *pipeline.Pipeline, file, hashstring string) (*ClientLogger, error) {
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
	inputFile, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening logger fifo: %q", file)
	}
	logp.Info("Created new logger for %s", file)

	return &ClientLogger{logFile: inputFile, client: client, pipelineHash: hashstring}, nil
}

// Close closes the pipeline client and reader
func (cl *ClientLogger) Close() error {
	logp.Info("Closing ClientLogger")
	cl.client.Close()

	return cl.logFile.Close()
}

// ConsumeAndSendLogs reads from the FIFO file and sends to the pipeline client. This will block and should be called in its own goroutine
// TODO: Publish() can block, which is a problem. This whole thing should be two goroutines.
func (cl *ClientLogger) ConsumeAndSendLogs() {
	reader := pb.NewUint32DelimitedReader(cl.logFile, binary.BigEndian, 2e6)

	publishWriter := make(chan logdriver.LogEntry, 500)

	go cl.publishLoop(publishWriter)
	// Clean up the reader after we're done
	defer func() {

		close(publishWriter)

		err := reader.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error closing FIFO reader: %s", err)
		}
	}()

	var log logdriver.LogEntry
	for {
		err := reader.ReadMsg(&log)
		if err != nil {
			if err == io.EOF || err == os.ErrClosed || strings.Contains(err.Error(), "file already closed") {
				cl.logFile.Close()
				return
			}
			// I am...not sure why we do this
			reader = pb.NewUint32DelimitedReader(cl.logFile, binary.BigEndian, 2e6)
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

		cl.client.Publish(beat.Event{
			Timestamp: time.Unix(0, entry.TimeNano),
			Fields: common.MapStr{
				"line": string(entry.Line),
			},
		})

	}

}

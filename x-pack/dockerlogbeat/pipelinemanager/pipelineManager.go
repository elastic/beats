// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/v8/x-pack/dockerlogbeat/pipereader"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	"github.com/docker/docker/daemon/logger/jsonfilelog"

	protoio "github.com/gogo/protobuf/io"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/publisher/pipeline"
)

// containerConfig is the common.Config unpacking type
type containerConfig struct {
	Pipeline pipeline.Config        `config:"pipeline"`
	Output   common.ConfigNamespace `config:"output"`
}

// Pipeline represents a single pipeline and the count of associated clients
type Pipeline struct {
	pipeline *pipeline.Pipeline
	refCount int
}

// PipelineManager is a handler into the map of pipelines used by the plugin
type PipelineManager struct {
	mu     sync.Mutex
	Logger *logp.Logger
	// pipelines key: config hash
	pipelines map[uint64]*Pipeline
	// clients config: filepath
	clients map[string]*ClientLogger
	// Client Logger key: container hash
	clientLogger map[string]logger.Logger
	// logDirectory is the bindmount for local container logsd
	logDirectory string
	// destroyLogsOnStop indicates for the client to remove log files when a container stops
	destroyLogsOnStop bool
	// hostname of the docker host
	hostname string
}

// NewPipelineManager creates a new Pipeline map
func NewPipelineManager(logDestroy bool, hostname string) *PipelineManager {
	return &PipelineManager{
		Logger:            logp.NewLogger("PipelineManager"),
		pipelines:         make(map[uint64]*Pipeline),
		clients:           make(map[string]*ClientLogger),
		clientLogger:      make(map[string]logger.Logger),
		logDirectory:      "/var/log/docker/containers",
		destroyLogsOnStop: logDestroy,
		hostname:          hostname,
	}
}

// CloseClientWithFile closes the client with the associated file
func (pm *PipelineManager) CloseClientWithFile(file string) error {

	cl, err := pm.removeClient(file)
	if err != nil {
		return errors.Wrap(err, "Error removing client")
	}

	hash := cl.pipelineHash

	// remove the logger
	pm.removeLogger(cl.ContainerMeta)

	pm.Logger.Debugf("Closing Client first from pipelineManager")
	err = cl.Close()
	if err != nil {
		return errors.Wrap(err, "error closing client")
	}

	// if the pipeline is no longer in use, clean up
	pm.removePipelineIfNeeded(hash)

	return nil
}

// CreateClientWithConfig gets the pipeline linked to the given config, and creates a client
// If no pipeline for that config exists, it creates one.
func (pm *PipelineManager) CreateClientWithConfig(containerConfig ContainerOutputConfig, info logger.Info, file string) (*ClientLogger, error) {

	hashstring, err := hashstructure.Hash(containerConfig, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating config hash")
	}
	pipeline, err := pm.getOrCreatePipeline(containerConfig, hashstring)
	if err != nil {
		return nil, errors.Wrap(err, "error getting pipeline")
	}

	reader, err := pipereader.NewReaderFromPath(file)
	if err != nil {
		return nil, errors.Wrap(err, "error creating reader for docker log stream")
	}

	// Why is this empty by default? What should be here? Who knows!
	if info.LogPath == "" {
		info.LogPath = filepath.Join(pm.logDirectory, info.ContainerID, fmt.Sprintf("%s-json.log", info.ContainerID))
	}
	err = os.MkdirAll(filepath.Dir(info.LogPath), 0755)
	if err != nil {
		return nil, errors.Wrap(err, "error creating directory for local logs")
	}
	// set a default log size
	if _, ok := info.Config["max-size"]; !ok {
		info.Config["max-size"] = "10M"
	}
	// set a default log count
	if _, ok := info.Config["max-file"]; !ok {
		info.Config["max-file"] = "5"
	}

	localLog, err := jsonfilelog.New(info)
	if err != nil {
		return nil, errors.Wrap(err, "error creating local log")
	}

	//actually get to crafting the new client.
	cl, err := newClientFromPipeline(pipeline.pipeline, reader, hashstring, info, localLog, pm.hostname)
	if err != nil {
		return nil, errors.Wrap(err, "error creating client")
	}

	pm.registerClient(cl, hashstring, file)
	pm.registerLogger(localLog, info)
	return cl, nil
}

// CreateReaderForContainer responds to docker logs requests to pull local logs from the json logger
func (pm *PipelineManager) CreateReaderForContainer(info logger.Info, config logger.ReadConfig) (io.ReadCloser, error) {
	logObject, exists := pm.getLogger(info)
	if !exists {
		return nil, fmt.Errorf("Could not find logger for %s", info.ContainerID)
	}
	pipeReader, pipeWriter := io.Pipe()
	logReader, ok := logObject.(logger.LogReader)
	if !ok {
		return nil, fmt.Errorf("logger does not support reading")
	}

	go func() {
		watcher := logReader.ReadLogs(config)

		enc := protoio.NewUint32DelimitedWriter(pipeWriter, binary.BigEndian)
		defer enc.Close()
		defer watcher.ConsumerGone()
		var rawLog logdriver.LogEntry
		for {
			select {
			case msg, ok := <-watcher.Msg:
				if !ok {
					pipeWriter.Close()
					return
				}
				rawLog.Line = msg.Line
				rawLog.Partial = msg.PLogMetaData != nil
				rawLog.TimeNano = msg.Timestamp.UnixNano()
				rawLog.Source = msg.Source

				if err := enc.WriteMsg(&rawLog); err != nil {
					pipeWriter.CloseWithError(err)
					return
				}

			case err := <-watcher.Err:
				pipeWriter.CloseWithError(err)
				return

			}
		}

	}()

	return pipeReader, nil
}

//===================
// Private methods

// checkAndCreatePipeline performs the pipeline check and creation as one atomic operation
// It will either return a new pipeline, or an existing one from the pipeline map
func (pm *PipelineManager) getOrCreatePipeline(logOptsConfig ContainerOutputConfig, hash uint64) (*Pipeline, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var pipeline *Pipeline
	var err error
	pipeline, test := pm.pipelines[hash]
	if !test {
		pipeline, err = loadNewPipeline(logOptsConfig, pm.hostname, pm.Logger)
		if err != nil {
			return nil, errors.Wrap(err, "error loading pipeline")
		}
		pm.pipelines[hash] = pipeline
	}

	return pipeline, nil
}

// getClient gets a pipeline client based on a file handle
func (pm *PipelineManager) getClient(file string) (*ClientLogger, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	cli, exists := pm.clients[file]
	return cli, exists
}

func (pm *PipelineManager) getLogger(info logger.Info) (logger.Logger, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	logger, exists := pm.clientLogger[info.ContainerID]
	return logger, exists
}

// removePipeline removes a pipeline from the manager if it's refcount is zero.
func (pm *PipelineManager) removePipelineIfNeeded(hash uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	//if the pipeline is no longer in use, clean up
	if pm.pipelines[hash].refCount < 1 {
		pipeline := pm.pipelines[hash].pipeline
		delete(pm.pipelines, hash)
		//pipelines must be closed after clients
		//Just do this here, since the caller doesn't know if we need to close the libbeat pipeline
		pm.Logger.Debugf("Pipeline closing from removePipelineIfNeeded")
		err := pipeline.Close()
		if err != nil {
			pm.Logger.Errorf("Error closing pipeline: %s", err)
		}
	}
}

// registerClient registers a new client with the manager. Up to the caller to  actually close the libbeat client
func (pm *PipelineManager) registerClient(cl *ClientLogger, hash uint64, clientFile string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.clients[clientFile] = cl
	pm.pipelines[hash].refCount++
}

// registerLogger registers a local logger used for reading back logs
func (pm *PipelineManager) registerLogger(log logger.Logger, info logger.Info) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.clientLogger[info.ContainerID] = log
}

// removeLogger removes a logging instace
func (pm *PipelineManager) removeLogger(info logger.Info) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	logger, exists := pm.clientLogger[info.ContainerID]
	if !exists {
		return
	}
	logger.Close()
	delete(pm.clientLogger, info.ContainerID)
	if pm.destroyLogsOnStop {
		pm.removeLogFile(info.ContainerID)
	}
}

// removeLogFile removes a log file for a given container. Disabled by default.
func (pm *PipelineManager) removeLogFile(id string) error {
	toRemove := filepath.Join(pm.logDirectory, id)

	return os.Remove(toRemove)
}

// removeClient deregisters a client
func (pm *PipelineManager) removeClient(file string) (*ClientLogger, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cl, ok := pm.clients[file]
	if !ok {
		return nil, fmt.Errorf("No client for file %s", file)
	}

	// deincrement the ref count
	hash := cl.pipelineHash
	pm.pipelines[hash].refCount--
	delete(pm.clients, file)

	return cl, nil
}

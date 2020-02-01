// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/x-pack/dockerlogbeat/pipereader"

	"github.com/docker/docker/daemon/logger"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
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
	pipelines map[string]*Pipeline
	// clients config: filepath
	clients map[string]*ClientLogger
}

// NewPipelineManager creates a new Pipeline map
func NewPipelineManager(logCfg *common.Config) *PipelineManager {
	return &PipelineManager{
		Logger:    logp.NewLogger("PipelineManager"),
		pipelines: make(map[string]*Pipeline),
		clients:   make(map[string]*ClientLogger),
	}
}

// CloseClientWithFile closes the client with the associated file
func (pm *PipelineManager) CloseClientWithFile(file string) error {

	cl, err := pm.removeClient(file)
	if err != nil {
		return errors.Wrap(err, "Error removing client")
	}

	hash := cl.pipelineHash

	pm.Logger.Debugf("Closing Client first from pipelineManager")
	err = cl.Close()
	if err != nil {
		return errors.Wrap(err, "error closing client")
	}

	//if the pipeline is no longer in use, clean up
	pm.removePipelineIfNeeded(hash)

	return nil
}

// CreateClientWithConfig gets the pipeline linked to the given config, and creates a client
// If no pipeline for that config exists, it creates one.
func (pm *PipelineManager) CreateClientWithConfig(containerConfig logger.Info, file string) (*ClientLogger, error) {

	hashstring := makeConfigHash(containerConfig.Config)
	pipeline, err := pm.getOrCreatePipeline(containerConfig.Config, file, hashstring)
	if err != nil {
		return nil, errors.Wrap(err, "error getting pipeline")
	}

	reader, err := pipereader.NewReaderFromPath(file)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	//actually get to crafting the new client.
	cl, err := newClientFromPipeline(pipeline.pipeline, reader, hashstring, containerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error creating client")
	}

	pm.registerClient(cl, hashstring, file)

	return cl, nil
}

//===================
// Private methods

// checkAndCreatePipeline performs the pipeline check and creation as one atomic operation
// It will either return a new pipeline, or an existing one from the pipeline map
func (pm *PipelineManager) getOrCreatePipeline(logOptsConfig map[string]string, file string, hashstring string) (*Pipeline, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var pipeline *Pipeline
	var err error
	pipeline, test := pm.pipelines[hashstring]
	if !test {
		pipeline, err = loadNewPipeline(logOptsConfig, file, pm.Logger)
		if err != nil {
			return nil, errors.Wrap(err, "error loading pipeline")
		}
		pm.pipelines[hashstring] = pipeline
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

// removePipeline removes a pipeline from the manager if it's refcount is zero.
func (pm *PipelineManager) removePipelineIfNeeded(hash string) {
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
func (pm *PipelineManager) registerClient(cl *ClientLogger, hashstring, clientFile string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.clients[clientFile] = cl
	pm.pipelines[hashstring].refCount++
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

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
	"github.com/pkg/errors"
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
		Logger: logp.NewLogger("dockerlogbeat"),
		//mu:        new(sync.Mutex),
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
func (pm *PipelineManager) CreateClientWithConfig(logOptsConfig map[string]string, file string) (*ClientLogger, error) {

	hashstring := makeConfigHash(logOptsConfig)

	//If we don't have an existing pipeline for this hash, make one
	exists := pm.checkIfHashExists(logOptsConfig)
	var pipeline *Pipeline
	var err error
	if !exists {
		pipeline, err = loadNewPipeline(logOptsConfig, file, pm.Logger)
		if err != nil {
			return nil, errors.Wrap(err, "error loading pipeline")
		}
		pm.registerPipeline(pipeline, hashstring)
	} else {
		pipeline, _ = pm.getPipeline(hashstring)
	}

	//actually get to crafting the new client.
	cl, err := newClientFromPipeline(pipeline.pipeline, file, hashstring)
	if err != nil {
		return nil, err
	}

	pm.registerClient(cl, hashstring, file)

	return cl, nil
}

//===================
// Private methods

// getPipeline gets a pipeline based on a confighash
func (pm *PipelineManager) getPipeline(hashstring string) (*Pipeline, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pipeline, exists := pm.pipelines[hashstring]
	return pipeline, exists
}

// getClient gets a pipeline client based on a file handle
func (pm *PipelineManager) getClient(file string) (*ClientLogger, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	cli, exists := pm.clients[file]
	return cli, exists
}

// checkIfHashExists is a short atomic function to see if a pipeline alread exists inside the PM. Thread-safe.
func (pm *PipelineManager) checkIfHashExists(logOptsConfig map[string]string) bool {
	hashstring := makeConfigHash(logOptsConfig)
	pm.mu.Lock()
	defer pm.mu.Unlock()
	_, test := pm.pipelines[hashstring]
	if test {
		return true
	}
	return false
}

// registerPipeline is a small atomic function that registers a new pipeline with the managers
// TODO: What happens if we try to register a pipeline that already exists? Which pipeline "wins"?
func (pm *PipelineManager) registerPipeline(pipeline *Pipeline, hashstring string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pipelines[hashstring] = pipeline

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
		go func() {
			pm.Logger.Debugf("Pipeline closing from removePipelineIfNeeded")
			pipeline.Close()
		}()
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

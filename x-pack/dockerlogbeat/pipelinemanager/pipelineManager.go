// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
	"github.com/elastic/beats/libbeat/publisher/processing"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
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
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cl, ok := pm.clients[file]
	if !ok {
		return fmt.Errorf("No client for file %s", file)
	}

	// deincrement the ref count
	hash := cl.pipelineHash
	pm.pipelines[hash].refCount--

	pm.Logger.Infof("Closing Client first from pipelineManager")
	err := cl.Close()
	if err != nil {
		return errors.Wrap(err, "error closing client")
	}

	if pm.pipelines[hash].refCount < 1 {
		pm.Logger.Infof("Pipeline  closing")
		pm.pipelines[hash].pipeline.Close()
		delete(pm.pipelines, hash)
	}

	return nil
}

// CreateClientWithConfig gets the pipeline linked to the given config, and creates a client
// If no pipeline for that config exists, it creates one.
func (pm *PipelineManager) CreateClientWithConfig(logOptsConfig map[string]string, file string) (*ClientLogger, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// check to see if we have a pipeline
	var err error
	hashstring := makeConfigHash(logOptsConfig)

	err = pm.makePipeline(logOptsConfig, file, hashstring)
	if err != nil {
		return nil, err
	}

	pipeline := pm.pipelines[hashstring]

	cl, err := newClientFromPipeline(pipeline.pipeline, file, hashstring)
	if err != nil {
		return nil, err
	}
	pm.clients[file] = cl
	pm.pipelines[hashstring].refCount++
	// before we finish, prune the client list async

	return cl, nil
}

// a wrapper for various public functions to create pipelines
// assumes we have a mutex lock. If the pipeline exists, this does nothing.
func (pm *PipelineManager) makePipeline(logOptsConfig map[string]string, name, hashstring string) error {
	_, test := pm.pipelines[hashstring]
	if test {
		return nil
	}

	pipeline, err := loadNewPipeline(logOptsConfig, name, pm.Logger)
	if err != nil {
		return errors.Wrap(err, "error loading pipeline")
	}
	pm.pipelines[hashstring] = &Pipeline{pipeline: pipeline, refCount: 0}
	return nil
}

// makeConfigHash is the helper function that turns a user config into a hash
func makeConfigHash(cfg map[string]string) string {
	var hashString string
	var orderedVal []string

	for _, val := range cfg {
		orderedVal = append(orderedVal, val)
	}

	sort.Strings(orderedVal)

	for _, val := range orderedVal {
		hashString = hashString + val
	}

	sum := sha1.Sum([]byte(hashString))

	return string(sum[:])
}

// load pipeline starts up a new pipeline with the given config
func loadNewPipeline(logOptsConfig map[string]string, name string, log *logp.Logger) (*pipeline.Pipeline, error) {
	info := beat.Info{
		Beat:     "dockerlogbeat",
		Version:  "0",
		Name:     name,
		Hostname: "dockerbeat.test",
	}

	newCfg, err := parseCfgKeys(logOptsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing config keys")
	}

	cfg, err := common.NewConfigFrom(newCfg)
	if err != nil {
		return nil, err
	}
	config := containerConfig{}
	err = cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("unpacking config failed: %v", err)
	}

	processing, err := processing.MakeDefaultSupport(false)(info, log, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error in MakeDefaultSupport")
	}

	pipelineCfg := pipeline.Config{}
	err = cfg.Unpack(&pipelineCfg)
	if err != nil {
		return nil, errors.Wrap(err, "error unpacking pipeline config")
	}

	idx, err := idxmgmt.DefaultSupport(log, info, config.Output.Config())
	if err != nil {
		return nil, errors.Wrap(err, "error making index manager")
	}

	pipeline, err := pipeline.Load(info,
		pipeline.Monitors{
			Metrics:   nil,
			Telemetry: nil,
			Logger:    log,
		},
		pipelineCfg,
		processing,
		func(stat outputs.Observer) (string, outputs.Group, error) {
			cfg := config.Output
			out, err := outputs.Load(idx, info, stat, cfg.Name(), cfg.Config())
			return cfg.Name(), out, err
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "error in pipeline.Load")
	}

	return pipeline, nil
}

// parseCfgKeys helpfully parses the values in the map, so users can specify yml structures.
func parseCfgKeys(cfg map[string]string) (map[string]interface{}, error) {

	outMap := make(map[string]interface{})

	for cfgKey, strVal := range cfg {
		var parsed interface{}
		if err := yaml.Unmarshal([]byte(strVal), &parsed); err != nil {
			return nil, err
		}
		outMap[cfgKey] = parsed
	}

	return outMap, nil
}

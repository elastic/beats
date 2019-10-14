// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"crypto/sha1"
	"fmt"
	"sort"

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
func loadNewPipeline(logOptsConfig map[string]string, name string, log *logp.Logger) (*Pipeline, error) {
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

	return &Pipeline{pipeline: pipeline, refCount: 0}, nil
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

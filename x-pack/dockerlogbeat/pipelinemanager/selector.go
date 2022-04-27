// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/elastic-agent-libs/config"
)

// IdxSupport is a supporter type used by libbeat to manage index support
type IdxSupport struct {
	defaultIndex string
	beatInfo     beat.Info
}

// newIndexSupporter returns an index support type for use with outputs.Load
func newIndexSupporter(info beat.Info) *IdxSupport {
	return &IdxSupport{
		beatInfo:     info,
		defaultIndex: fmt.Sprintf("%v-%v-%%{+yyyy.MM.dd}", info.IndexPrefix, info.Version),
	}
}

// BuildSelector implements the IndexManager interface
func (s *IdxSupport) BuildSelector(cfg *config.C) (outputs.IndexSelector, error) {
	//copy the config object we get before we send it to the BuildSelector
	bsCfg := config.NewConfig()
	if cfg.HasField("indicies") {
		sub, err := cfg.Child("indices", -1)
		if err != nil {
			return nil, errors.Wrap(err, "error getting indicies field")
		}
		bsCfg.SetChild("indices", -1, sub)
	}

	var err error
	var suppliedIndex string
	if cfg.HasField("index") {
		suppliedIndex, err = cfg.String("index", -1)
		if err != nil {
			return nil, err
		}
	}

	if suppliedIndex == "" {
		suppliedIndex = s.defaultIndex
	}
	bsCfg.SetString("index", -1, suppliedIndex)

	buildSettings := outil.Settings{
		Key:              "index",
		MultiKey:         "indices",
		EnableSingleOnly: true,
		FailEmpty:        true,
	}

	indexSel, err := outil.BuildSelectorFromConfig(bsCfg, buildSettings)
	if err != nil {
		return nil, errors.Wrap(err, "error creating build Selector")
	}

	return indexSel, nil
}

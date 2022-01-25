// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"fmt"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/cfgutil"
)

// Loader is used to load configuration from the paths
// including appending multiple input configurations.
type Loader struct {
	logger       *logger.Logger
	inputsFolder string
}

// NewLoader creates a new Loader instance to load configuration
// files from different paths.
func NewLoader(logger *logger.Logger, inputsFolder string) *Loader {
	return &Loader{logger: logger, inputsFolder: inputsFolder}
}

// Load iterates over the list of files and loads the confguration from them.
// If a configuration file is under the folder set in `agent.config.inputs.path`
// it is appended to a list. If it is a regular config file, it is merged into
// the result config. The list of input configurations is merged into the result
// last.
func (l *Loader) Load(files []string) (*Config, error) {
	inputsList := make([]*ucfg.Config, 0)
	merger := cfgutil.NewCollector(nil)
	for _, f := range files {
		cfg, err := LoadFile(f)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration file '%s': %w", f, err)
		}
		l.logger.Debug("Loaded configuration from %s", f)
		if l.isFileUnderInputsFolder(f) {
			inp, err := getInput(cfg)
			if err != nil {
				return nil, fmt.Errorf("cannot get configuration from '%s': %w", f, err)
			}
			inputsList = append(inputsList, inp...)
			l.logger.Debug("Loaded %s input(s) from configuration from %s", len(inp), f)
		} else {
			if err := merger.Add(cfg.access(), err); err != nil {
				return nil, fmt.Errorf("failed to merge configuration file '%s' to existing one: %w", f, err)
			}
			l.logger.Debug("Merged configuration from %s into result", f)
		}
	}
	config := merger.Config()

	// if there is no input configuration, return what we have collected.
	if len(inputsList) == 0 {
		l.logger.Debug("Merged all configuration files from %v, no external input files", files)
		return newConfigFrom(config), nil
	}

	// merge inputs sections from the last standalone configuration
	// file and all files from the inputs folder
	start := 0
	if config.HasField("inputs") {
		var err error
		start, err = config.CountField("inputs")
		if err != nil {
			return nil, fmt.Errorf("failed to count the number of inputs in the configuration: %w", err)
		}
	}
	for i, ll := range inputsList {
		if err := config.SetChild("inputs", start+i, ll); err != nil {
			return nil, fmt.Errorf("failed to add inputs to result configuration: %w", err)
		}
	}

	l.logger.Debug("Merged all configuration files from %v, with external input files", files)
	return newConfigFrom(config), nil
}

func getInput(c *Config) ([]*ucfg.Config, error) {
	tmpConfig := struct {
		Inputs []*ucfg.Config `config:"inputs"`
	}{make([]*ucfg.Config, 0)}

	if err := c.Unpack(&tmpConfig); err != nil {
		return nil, fmt.Errorf("failed to parse inputs section from configuration: %w", err)
	}
	return tmpConfig.Inputs, nil
}

func (l *Loader) isFileUnderInputsFolder(f string) bool {
	if matches, err := filepath.Match(l.inputsFolder, f); !matches || err != nil {
		return false
	}
	return true
}

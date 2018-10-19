// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/x-pack/beatless/config"
	"github.com/elastic/beats/x-pack/beatless/core/bundle"
)

// Package size limits for function providers, we should be a lot under this limit but
// adding a check to make sure we never go over.
const packageCompressedLimit = 50 * 1000 * 1000    // 50MB
const packageUncompressedLimit = 250 * 1000 * 1000 // 250MB

func rawYaml() ([]byte, error) {
	// Load the configuration file from disk with all the settings,
	// the function takes care of using -c.
	rawConfig, err := cfgfile.Load("", config.ConfigOverrides)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, err
	}

	res, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MakeZip creates a zip from the the current artifacts and the currently available configuration.
func MakeZip() ([]byte, error) {
	rawConfig, err := rawYaml()
	if err != nil {
		return nil, err
	}
	bundle := bundle.NewZipWithLimits(
		packageUncompressedLimit,
		packageCompressedLimit,
		&bundle.MemoryFile{Path: "beatless.yml", Raw: rawConfig, FileMode: 0766},
		&bundle.LocalFile{Path: "pkg/beatless", FileMode: 0755},
	)

	content, err := bundle.Bytes()
	if err != nil {
		return nil, err
	}
	return content, nil
}

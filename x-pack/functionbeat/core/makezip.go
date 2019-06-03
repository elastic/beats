// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/keystore"
	"github.com/elastic/beats/x-pack/functionbeat/config"
	"github.com/elastic/beats/x-pack/functionbeat/core/bundle"
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

	resources := []bundle.Resource{
		&bundle.MemoryFile{Path: "functionbeat.yml", Raw: rawConfig, FileMode: 0766},
		&bundle.LocalFile{Path: "pkg/functionbeat", FileMode: 0755},
	}

	rawKeystore, err := keystoreRaw()
	if err != nil {
		return nil, err
	}

	if len(rawKeystore) > 0 {
		resources = append(resources, &bundle.MemoryFile{
			Path:     "data/functionbeat.keystore",
			Raw:      rawKeystore,
			FileMode: 0600,
		})
	}

	bundle := bundle.NewZipWithLimits(
		packageUncompressedLimit,
		packageCompressedLimit,
		resources...)

	content, err := bundle.Bytes()
	if err != nil {
		return nil, err
	}
	return content, nil
}

func keystoreRaw() ([]byte, error) {
	cfg, err := cfgfile.Load("", common.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %v", err)
	}

	store, err := instance.LoadKeystore(cfg, "functionbeat")
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load the keystore for packaging")
	}

	packager, ok := store.(keystore.Packager)
	if !ok {
		return nil, fmt.Errorf("the configured keystore cannot be packaged")
	}

	return packager.Package()
}

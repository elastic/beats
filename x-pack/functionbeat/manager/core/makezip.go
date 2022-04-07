// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/cfgfile"
	"github.com/elastic/beats/v8/libbeat/cmd/instance"
	"github.com/elastic/beats/v8/libbeat/keystore"
	"github.com/elastic/beats/v8/x-pack/functionbeat/config"
	"github.com/elastic/beats/v8/x-pack/functionbeat/manager/core/bundle"
)

func rawYaml() ([]byte, error) {
	// Load the configuration file from disk with all the settings,
	// the function takes care of using -c.
	rawConfig, err := cfgfile.Load("", config.Overrides)
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
func MakeZip(
	packageUncompressedLimit int64,
	packageCompressedLimit int64,
	providerResources []bundle.Resource,
) ([]byte, error) {
	if len(providerResources) == 0 {
		return nil, fmt.Errorf("no provider specific resources are set")
	}

	rawConfig, err := rawYaml()
	if err != nil {
		return nil, err
	}

	resources := append(
		providerResources,
		&bundle.MemoryFile{Path: "functionbeat.yml", Raw: rawConfig, FileMode: 0766},
	)

	resources, err = addKeystoreIfConfigured(resources)
	if err != nil {
		return nil, err
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

func addKeystoreIfConfigured(resources []bundle.Resource) ([]bundle.Resource, error) {
	ksPackager, err := keystorePackager()
	if err != nil {
		return nil, err
	}

	rawKeystore, err := ksPackager.Package()
	if err != nil {
		return nil, err
	}

	if len(rawKeystore) > 0 {
		resources = append(resources, &bundle.MemoryFile{
			Path:     ksPackager.ConfiguredPath(),
			Raw:      rawKeystore,
			FileMode: 0600,
		})
	}

	return resources, nil
}

func keystorePackager() (keystore.Packager, error) {
	cfg, err := cfgfile.Load("", config.Overrides)
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

	return packager, nil
}

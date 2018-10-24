// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"io/ioutil"
	"os"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/x-pack/libbeat/management/api"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Cache keeps a copy of configs provided by Kibana, it's used when Kibana is down
type Cache struct {
	// ConfigOK is true if last config update was successful
	ConfigOK bool
	Configs  api.ConfigBlocks
}

// Load settings from its source file
func (c *Cache) Load() error {
	path := paths.Resolve(paths.Data, "management.yml")
	config, err := common.LoadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File is not present, beat is not enrolled
			return nil
		}
		return err
	}

	if err = config.Unpack(&c); err != nil {
		return err
	}

	return nil
}

// Save settings to management.yml file
func (c *Cache) Save() error {
	path := paths.Resolve(paths.Data, "management.yml")

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// write temporary file first
	tempFile := path + ".new"
	if err := ioutil.WriteFile(tempFile, data, 0600); err != nil {
		return errors.Wrap(err, "failed to store central management settings")
	}

	// move temporary file into final location
	return file.SafeFileRotate(path, tempFile)
}
